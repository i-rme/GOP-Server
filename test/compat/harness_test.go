// Package compat is a black-box compatibility suite for GOP Server.
//
// It starts a server binary in a throwaway copy of the site (config, public,
// private, go.mod), replays the requests defined in cases_test.go, normalises
// anything that changes from run to run, and compares the result with the
// golden files under testdata/golden. See README.md for how to run it.
package compat

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

var update = flag.Bool("update", false, "rewrite golden files with the current responses")

const (
	userAgent     = "gop-compat/1"
	startTimeout  = 60 * time.Second
	clientTimeout = 3 * time.Minute
)

// scenario is one server start with its own config overrides and cases.
type scenario struct {
	Name   string
	Config map[string]interface{} // merged over config/configuration.json
	Cases  []testCase
}

// testCase is a named sequence of requests recorded into one golden file.
// Steps run in order against the same server, so cache state carries over.
type testCase struct {
	Name      string
	Steps     []step
	Normalize []rule // applied before the global rules
	Skip      string // reason, when the case is known not to be runnable
}

type step struct {
	Method    string
	Path      string // may contain {{var}} placeholders filled by Extract
	Header    map[string]string
	Form      url.Values // urlencoded body
	Multipart *multipartBody
	Extract   []extract // save values from this response for later steps
	ListDir   string    // after the request, record the files in this site-relative dir
}

type multipartBody struct {
	Fields    map[string]string
	FileField string // may contain {{var}}
	FileName  string
	FileData  string
}

// extract saves the first submatch of Pattern, found in the raw body or in a
// response header, as a variable usable in later steps of the same case.
type extract struct {
	Var     string
	Header  string // empty means the body
	Pattern string
}

type rule struct {
	re   *regexp.Regexp
	repl string
}

func re(pattern, repl string) rule { return rule{regexp.MustCompile(pattern), repl} }

// droppedHeaders vary with timing or transfer details, not with behaviour.
var droppedHeaders = map[string]bool{
	"Date":              true,
	"Content-Length":    true,
	"Transfer-Encoding": true,
}

// globalRules apply to every header value and text body, after the
// literal substitutions of the site root and host.
var globalRules = []rule{
	re(`REMOTE_PORT:\d+`, "REMOTE_PORT:{{REMOTE_PORT}}"),
	re(`(<th>Port</th><td>)\d+`, "${1}{{PORT}}"),
	re(`[0-9a-f]{64}`, "{{SHA256}}"),
	re(`temp-\d+\.go`, "temp-{{N}}.go"),
	re(`go-build\d+`, "go-build{{N}}"),
}

// server is a running GOP Server under test.
type server struct {
	root string // site copy the server runs in
	host string // 127.0.0.1:port
	log  *lockedBuffer
	cmd  *exec.Cmd
}

// lockedBuffer collects server output, which exec copies from another goroutine.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestCompat(t *testing.T) {
	bin := serverBinary(t)
	site := siteRoot(t)

	for _, sc := range scenarios() {
		sc := sc
		t.Run(sc.Name, func(t *testing.T) {
			srv := startServer(t, bin, site, sc.Config)
			client := &http.Client{
				Timeout:       clientTimeout,
				Transport:     &http.Transport{DisableCompression: true, DisableKeepAlives: true},
				CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
			}
			for _, tc := range sc.Cases {
				tc := tc
				t.Run(tc.Name, func(t *testing.T) {
					if tc.Skip != "" {
						t.Skip(tc.Skip)
					}
					got := runCase(t, client, srv, tc)
					compareGolden(t, filepath.Join("testdata", "golden", sc.Name, tc.Name+".golden"), got)
				})
			}
			if t.Failed() {
				t.Logf("server output (last 4 KB):\n%s", tail(srv.log.String(), 4096))
			}
		})
	}
}

// serverBinary returns GOP_SERVER_BIN, or builds ./src from the repository root.
func serverBinary(t *testing.T) string {
	if bin := os.Getenv("GOP_SERVER_BIN"); bin != "" {
		abs, err := filepath.Abs(bin)
		if err != nil {
			t.Fatal(err)
		}
		return abs
	}
	bin := filepath.Join(t.TempDir(), "gop-server")
	cmd := exec.Command("go", "build", "-o", bin, "./src")
	cmd.Dir = siteRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building server (set GOP_SERVER_BIN to use a prebuilt binary): %v\n%s", err, out)
	}
	return bin
}

// siteRoot is the repository root, which holds config/, public/, private/ and go.mod.
func siteRoot(t *testing.T) string {
	abs, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func startServer(t *testing.T, bin, site string, overrides map[string]interface{}) *server {
	t.Helper()
	work := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(work); err == nil {
		work = resolved
	}
	for _, name := range []string{"public", "private", "go.mod", "go.sum"} {
		if err := copyTree(filepath.Join(site, name), filepath.Join(work, name)); err != nil {
			t.Fatalf("copying %s: %v", name, err)
		}
	}

	cfg := map[string]interface{}{}
	raw, err := os.ReadFile(filepath.Join(site, "config", "configuration.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parsing configuration.json: %v", err)
	}
	port := freePort(t)
	cfg["Address"] = "127.0.0.1"
	cfg["Port"] = port
	for k, v := range overrides {
		cfg[k] = v
	}

	// Extra scripts that exercise helpers the examples don't cover.
	docRoot, _ := cfg["DocumentRoot"].(string)
	if err := copyTree(filepath.Join("testdata", "fixtures"), filepath.Join(work, docRoot)); err != nil {
		t.Fatalf("copying fixtures: %v", err)
	}
	for _, key := range []string{"LogRoot", "UploadsRoot"} {
		if dir, _ := cfg[key].(string); dir != "" {
			if err := os.MkdirAll(filepath.Join(work, dir), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.MkdirAll(filepath.Join(work, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(filepath.Join(work, "config", "configuration.json"), out, 0o644); err != nil {
		t.Fatal(err)
	}

	srv := &server{root: work, host: fmt.Sprintf("127.0.0.1:%d", port), log: &lockedBuffer{}}
	srv.cmd = exec.Command(bin)
	srv.cmd.Dir = work
	srv.cmd.Stdout = srv.log
	srv.cmd.Stderr = srv.log
	setProcessGroup(srv.cmd)
	if err := srv.cmd.Start(); err != nil {
		t.Fatalf("starting %s: %v", bin, err)
	}
	exited := make(chan struct{})
	go func() { srv.cmd.Wait(); close(exited) }()
	t.Cleanup(func() {
		killProcessGroup(srv.cmd)
		<-exited
	})

	deadline := time.Now().Add(startTimeout)
	for {
		conn, err := net.DialTimeout("tcp", srv.host, time.Second)
		if err == nil {
			conn.Close()
			return srv
		}
		select {
		case <-exited:
			t.Fatalf("server exited before listening on %s:\n%s", srv.host, srv.log.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not listen on %s within %s:\n%s", srv.host, startTimeout, srv.log.String())
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func runCase(t *testing.T, client *http.Client, srv *server, tc testCase) string {
	vars := map[string]string{}
	var out strings.Builder
	for i, st := range tc.Steps {
		req := buildRequest(t, srv, st, vars)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("step %d %s %s: %v", i+1, req.Method, req.URL.RequestURI(), err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("step %d reading body: %v", i+1, err)
		}
		for _, ex := range st.Extract {
			src := string(body)
			if ex.Header != "" {
				src = strings.Join(resp.Header.Values(ex.Header), "\n")
			}
			m := regexp.MustCompile(ex.Pattern).FindStringSubmatch(src)
			if len(m) < 2 {
				t.Fatalf("step %d: %q not found for {{%s}} in %q", i+1, ex.Pattern, ex.Var, tail(src, 500))
			}
			vars[ex.Var] = m[1]
		}
		writeStep(&out, srv, tc, st, vars, req, resp, body)
		if st.ListDir != "" {
			writeDirListing(t, &out, srv, tc, st.ListDir)
		}
	}
	return out.String()
}

func buildRequest(t *testing.T, srv *server, st step, vars map[string]string) *http.Request {
	method := st.Method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	contentType := ""
	switch {
	case st.Form != nil:
		body = strings.NewReader(st.Form.Encode())
		contentType = "application/x-www-form-urlencoded"
	case st.Multipart != nil:
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		mw.SetBoundary("gop-compat-boundary")
		keys := sortedKeys(st.Multipart.Fields)
		for _, k := range keys {
			mw.WriteField(k, st.Multipart.Fields[k])
		}
		if st.Multipart.FileField != "" {
			fw, err := mw.CreateFormFile(expand(st.Multipart.FileField, vars), st.Multipart.FileName)
			if err != nil {
				t.Fatal(err)
			}
			io.WriteString(fw, st.Multipart.FileData)
		}
		mw.Close()
		body = &buf
		contentType = mw.FormDataContentType()
	}
	req, err := http.NewRequest(method, "http://"+srv.host+expand(st.Path, vars), body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", userAgent)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range st.Header {
		req.Header.Set(k, expand(v, vars))
	}
	return req
}

var placeholder = regexp.MustCompile(`\{\{(\w+)\}\}`)

func expand(s string, vars map[string]string) string {
	return placeholder.ReplaceAllStringFunc(s, func(m string) string {
		if v, ok := vars[m[2:len(m)-2]]; ok {
			return v
		}
		return m
	})
}

// writeStep renders one request/response pair in the golden format:
//
//	=== METHOD /path
//	> Request-Header: value
//	--- STATUS
//	Response-Header: value
//
//	body
func writeStep(out *strings.Builder, srv *server, tc testCase, st step, vars map[string]string, req *http.Request, resp *http.Response, body []byte) {
	norm := func(s string) string { return normalize(s, srv, tc.Normalize) }

	fmt.Fprintf(out, "=== %s %s\n", req.Method, norm(req.URL.RequestURI()))
	for _, k := range sortedKeys(st.Header) {
		fmt.Fprintf(out, "> %s: %s\n", k, norm(expand(st.Header[k], vars)))
	}
	if st.Form != nil {
		fmt.Fprintf(out, "> [form] %s\n", norm(st.Form.Encode()))
	}
	if mp := st.Multipart; mp != nil {
		for _, k := range sortedKeys(mp.Fields) {
			fmt.Fprintf(out, "> [multipart] %s=%s\n", k, mp.Fields[k])
		}
		if mp.FileField != "" {
			fmt.Fprintf(out, "> [multipart file] %s=%s (%d bytes)\n", norm(expand(mp.FileField, vars)), mp.FileName, len(mp.FileData))
		}
	}

	fmt.Fprintf(out, "--- %d\n", resp.StatusCode)
	keys := make([]string, 0, len(resp.Header))
	for k := range resp.Header {
		if !droppedHeaders[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range resp.Header[k] {
			if k == "Last-Modified" {
				v = "{{HTTP-DATE}}"
			}
			fmt.Fprintf(out, "%s: %s\n", k, norm(v))
		}
	}
	out.WriteString("\n")

	if !utf8.Valid(body) {
		fmt.Fprintf(out, "[binary body: %d bytes, sha256 %x]\n", len(body), sha256.Sum256(body))
		return
	}
	text := norm(string(body))
	out.WriteString(text)
	if !strings.HasSuffix(text, "\n") {
		out.WriteString("\n\\ no newline at end of body\n")
	}
}

func writeDirListing(t *testing.T, out *strings.Builder, srv *server, tc testCase, dir string) {
	entries, err := os.ReadDir(filepath.Join(srv.root, dir))
	if err != nil {
		t.Fatalf("listing %s: %v", dir, err)
	}
	fmt.Fprintf(out, "=== [files in %s]\n", dir)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		name := normalize(e.Name(), srv, tc.Normalize)
		fmt.Fprintf(out, "%s (%d bytes)\n", name, info.Size())
	}
}

func normalize(s string, srv *server, extra []rule) string {
	for _, r := range extra {
		s = r.re.ReplaceAllString(s, r.repl)
	}
	s = strings.ReplaceAll(s, srv.root, "{{ROOT}}")
	s = strings.ReplaceAll(s, srv.host, "{{HOST}}")
	for _, r := range globalRules {
		s = r.re.ReplaceAllString(s, r.repl)
	}
	return s
}

func compareGolden(t *testing.T, path, got string) {
	t.Helper()
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden file (run with -update to create it): %v", err)
	}
	if string(want) == got {
		return
	}
	if dir := os.Getenv("GOP_COMPAT_OUT"); dir != "" {
		actual := filepath.Join(dir, strings.TrimPrefix(path, filepath.Join("testdata", "golden")+string(filepath.Separator)))
		os.MkdirAll(filepath.Dir(actual), 0o755)
		os.WriteFile(actual, []byte(got), 0o644)
	}
	t.Errorf("response differs from %s\n%s", path, firstDiff(string(want), got))
}

// firstDiff shows the first differing line with a little context.
func firstDiff(want, got string) string {
	w := strings.Split(want, "\n")
	g := strings.Split(got, "\n")
	i := 0
	for i < len(w) && i < len(g) && w[i] == g[i] {
		i++
	}
	from := i - 3
	if from < 0 {
		from = 0
	}
	var b strings.Builder
	fmt.Fprintf(&b, "first difference at line %d:\n", i+1)
	for j := from; j < i; j++ {
		fmt.Fprintf(&b, "  %s\n", clip(w[j]))
	}
	for j := i; j < i+3 && j < len(w); j++ {
		fmt.Fprintf(&b, "- %s\n", clip(w[j]))
	}
	for j := i; j < i+3 && j < len(g); j++ {
		fmt.Fprintf(&b, "+ %s\n", clip(g[j]))
	}
	return b.String()
}

func clip(s string) string {
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}

func tail(s string, n int) string {
	if len(s) > n {
		return "…" + s[len(s)-n:]
	}
	return s
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}
