package handler

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/i-rme/gop-server/src/server/config"
	"github.com/i-rme/gop-server/src/server/logs"
)

// site builds a document root with the given files and returns a Handler serving it
func site(t *testing.T, files map[string]string, configure func(*config.Config)) *Handler {
	t.Helper()
	if testing.Short() {
		t.Skip("builds scripts with the go command")
	}
	base := t.TempDir()
	files["private/listing.gop"] = "<!go\nfmt.Print(\"listing of \", _SERVER[\"SCRIPT_PATH_RELATIVE\"])"
	files["private/denied.gop"] = "<!go\n_status_code(403)\nfmt.Print(\"denied\")"
	for name, content := range files {
		path := filepath.Join(base, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	c := config.Default()
	c.DocumentRoot = filepath.Join(base, "public")
	c.CacheRoot = filepath.Join(base, "cache")
	c.UploadsRoot = filepath.Join(base, "uploads")
	c.DirectoryListingScript = "/../private/listing.gop"
	c.IPDeniedScript = "/../private/denied.gop"
	if configure != nil {
		configure(c)
	}
	for _, dir := range []string{c.DocumentRoot, c.CacheRoot, c.UploadsRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	h, err := New(c, slog.New(slog.NewTextHandler(io.Discard, nil)), &logs.Stats{})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func get(h *Handler, target string, header ...string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", target, nil)
	for i := 0; i+1 < len(header); i += 2 {
		r.Header.Set(header[i], header[i+1])
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestRequestsStayInDocumentRoot(t *testing.T) {
	h := site(t, map[string]string{
		"public/ok.txt":      "ok",
		"private/secret.txt": "TOP SECRET",
	}, nil)
	root := h.config.DocumentRoot
	if err := os.Symlink(filepath.Join(root, "..", "private"), filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}

	if w := get(h, "/ok.txt"); w.Code != 200 || w.Body.String() != "ok" {
		t.Errorf("/ok.txt: %d %q", w.Code, w.Body)
	}
	for _, target := range []string{"/escape/secret.txt", "/escape/", "/../private/secret.txt", "/%2e%2e/private/secret.txt"} {
		if w := get(h, target); w.Code != http.StatusNotFound || strings.Contains(w.Body.String(), "TOP SECRET") {
			t.Errorf("%s: got %d, want 404", target, w.Code)
		}
	}
}

func TestDirectories(t *testing.T) {
	h := site(t, map[string]string{
		"public/v1.2/index.gop": "<!go\nfmt.Print(\"index\")",
		"public/plain/a.txt":    "",
	}, nil)
	if w := get(h, "/v1.2/"); w.Body.String() != "index" {
		t.Errorf("a directory with a dot in its name: %d %q", w.Code, w.Body)
	}
	if w := get(h, "/plain"); w.Body.String() != "listing of /plain" {
		t.Errorf("a directory without index: %d %q", w.Code, w.Body)
	}
}

func TestLongestAliasWins(t *testing.T) {
	h := site(t, map[string]string{
		"public/v1.gop": "<!go\nfmt.Print(\"v1 \", _SERVER[\"GOP_SELF\"])",
		"public/v2.gop": "<!go\nfmt.Print(\"v2 \", _SERVER[\"GOP_SELF\"])",
	}, func(c *config.Config) {
		c.Aliases = map[string]string{"/api/": "/v1.gop", "/api/v2/": "/v2.gop", "/api/v": "/v1.gop"}
	})
	for target, want := range map[string]string{"/api/x": "v1 v1.gop", "/api/v2/x": "v2 v2.gop", "/api/v3": "v1 v1.gop"} {
		if w := get(h, target); w.Body.String() != want {
			t.Errorf("%s: got %q, want %q", target, w.Body, want)
		}
	}
}

func TestDeniedClientsGetTheDeniedScript(t *testing.T) {
	h := site(t, map[string]string{"public/a.txt": "a"}, func(c *config.Config) {
		_, network, _ := net.ParseCIDR("192.0.2.0/24")
		c.DenyFrom = []string{"192.0.2.0/24"}
		c.DeniedNetworks = []*net.IPNet{network}
	})
	r := httptest.NewRequest("GET", "/a.txt", nil)
	r.RemoteAddr = "192.0.2.7:1234"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 || w.Body.String() != "denied" {
		t.Errorf("got %d %q", w.Code, w.Body)
	}
}

func TestScriptHeaders(t *testing.T) {
	h := site(t, map[string]string{
		"public/headers.gop": `<!go
_header("X-Good: yes")
_header("X-Bad: a\r\nInjected: 1")
_header("Bad Name: x")
_setcookie("a", "1")
_setcookie("b", "2")
_status_code(1000)
`,
	}, nil)
	w := get(h, "/headers.gop")
	if w.Header().Get("X-Good") != "yes" || w.Header().Get("X-Bad") != "" || w.Header().Get("Injected") != "" {
		t.Errorf("headers %v", w.Header())
	}
	if cookies := w.Header().Values("Set-Cookie"); len(cookies) != 2 {
		t.Errorf("Set-Cookie %v, want both cookies", cookies)
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("an invalid status gave %d", w.Code)
	}
}

func TestSessionAcrossRequests(t *testing.T) {
	h := site(t, map[string]string{
		"public/count.gop": "<!go\n_session_start()\n_setsession(\"n\", _SESSION[\"n\"]+\"x\")\nfmt.Print(_SESSION[\"n\"])",
		"public/end.gop":   "<!go\n_session_destroy()",
	}, nil)

	cookie := ""
	for _, want := range []string{"x", "xx", "xxx"} {
		w := get(h, "/count.gop", "Cookie", cookie)
		if w.Body.String() != want {
			t.Fatalf("got %q, want %q", w.Body, want)
		}
		cookie = strings.Split(w.Header().Get("Set-Cookie"), ";")[0]
	}
	if w := get(h, "/end.gop", "Cookie", cookie); !strings.Contains(w.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Errorf("_session_destroy sent %q", w.Header().Get("Set-Cookie"))
	}
}

func TestParameters(t *testing.T) {
	h := site(t, map[string]string{"public/p.gop": "<!go\nfmt.Print(_GET, _POST)"}, nil)
	r := httptest.NewRequest("POST", "/p.gop?q=1", strings.NewReader("key=posted"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Body.String() != "map[q:1] map[key:posted]" {
		t.Errorf("got %q", w.Body)
	}
}
