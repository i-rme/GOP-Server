package compat

import (
	"encoding/base64"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func scenarios() []scenario {
	return []scenario{
		{Name: "default", Cases: defaultCases()},
		{
			Name:   "denied",
			Config: map[string]interface{}{"DenyFrom": []string{"127.0.0.0/8"}},
			Cases: []testCase{
				{Name: "script", Steps: []step{get("/examples/gop_file.gop"), get("/examples/gop_file.gop")}},
				{Name: "static", Steps: []step{get("/examples/text.txt")}},
				{Name: "gop_info", Steps: []step{get("/gop-info/")}},
			},
		},
		{
			// A day-long window keeps the counter from resetting mid-test.
			Name: "ratelimit",
			Config: map[string]interface{}{
				"RateLimitEnabled": true,
				"RateLimitRate":    2,
				"RateLimitPeriod":  86400,
			},
			Cases: []testCase{
				{Name: "over_limit", Steps: []step{
					get("/examples/gop_file.gop"),
					get("/examples/gop_file.gop"),
					get("/examples/gop_file.gop"),
					get("/examples/gop_file.gop"),
					get("/examples/text.txt"),
				}},
			},
		},
	}
}

// twice requests a path two times, to record the cache MISS and then HIT.
func twice(path string) []step { return []step{get(path), get(path)} }

func get(path string) step { return step{Path: path} }

func withHeader(path string, header map[string]string) step {
	return step{Path: path, Header: header}
}

func method(m, path string) step { return step{Method: m, Path: path} }

func postForm(path string, form url.Values) step {
	return step{Method: "POST", Path: path, Form: form}
}

func basicAuth(user, pass string) map[string]string {
	return map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))}
}

func defaultCases() []testCase {
	return []testCase{
		// Site pages and directory handling.
		{Name: "root_index", Steps: twice("/")},
		{Name: "license", Steps: twice("/license.gop")},
		{Name: "page_go", Steps: twice("/page.go")},
		{Name: "examples_index", Steps: []step{get("/examples/"), get("/examples"), get("/examples/index.gop")}},
		{Name: "directory_listing", Steps: []step{
			get("/examples/directorylisting"),
			get("/examples/directorylisting/"),
			get("/examples/dependencies/"),
		}},

		// Every example script, first request (MISS) and repeat (HIT).
		{Name: "gop_file", Steps: twice("/examples/gop_file.gop")},
		{Name: "gop_file_minified", Steps: twice("/examples/gop_file_minified.gop")},
		{Name: "gop_file_import_syntax", Steps: twice("/examples/gop_file_import_syntax.gop")},
		{Name: "gop_file_with_functions", Steps: twice("/examples/gop_file_with_functions.gop")},
		{Name: "gop_file_with_included_functions", Steps: twice("/examples/gop_file_with_included_functions.gop")},
		{Name: "gop_file_require", Steps: twice("/examples/gop_file_require.gop")},
		{Name: "gop_content_type", Steps: twice("/examples/gop_content_type.gop")},
		{Name: "gop_file_with_headers", Steps: twice("/examples/gop_file_with_headers.gop")},
		{Name: "go_file", Steps: twice("/examples/go_file.go")},
		{
			Name:      "go_file_with_functions",
			Steps:     twice("/examples/go_file_with_functions.go"),
			Normalize: []rule{re(`-> \d+`, "-> {{RANDOM}}")},
		},
		{
			Name:      "for_loop_100000",
			Steps:     twice("/examples/for_loop_100000.gop"),
			Normalize: []rule{re(`(?m)^\d+99999$`, "{{RANDOM_START}}...99999")},
		},
		{
			Name:      "gop_unprivileged_user",
			Steps:     twice("/examples/gop_unprivileged_user.gop"),
			Normalize: []rule{re(`(?m)\A[a-z_][a-z0-9_.-]*\n\z`, "{{USERNAME}}\n")},
		},
		{Name: "gop_mysql", Steps: twice("/examples/gop_mysql.gop")},

		// Example scripts that fail on purpose.
		{Name: "errors_gop_file_empty", Steps: twice("/examples/errors/gop_file_empty.gop")},
		{Name: "errors_gop_file_syntax_error", Steps: twice("/examples/errors/gop_file_syntax_error.gop")},
		{Name: "errors_gop_without_import", Steps: twice("/examples/errors/gop_without_import.gop")},
		{Name: "errors_gop_without_tag", Steps: twice("/examples/errors/gop_without_tag.gop")},
		{Name: "errors_go_file_empty", Steps: twice("/examples/errors/go_file_empty.go")},
		{Name: "errors_go_file_without_import", Steps: twice("/examples/errors/go_file_without_import.go")},
		{Name: "errors_gop_file_while_true", Steps: twice("/examples/errors/gop_file_while_true.gop")},
		{Name: "errors_go_file_while_true", Steps: twice("/examples/errors/go_file_while_true.go")},

		// GET and POST parameters.
		{Name: "get_parameters", Steps: []step{
			get("/examples/gop_file_get_parameters.gop"),
			get("/examples/gop_file_get_parameters.gop?key=value"),
			get("/examples/gop_file_get_parameters.gop?key=first&key=second&empty=&space=hello%20world&amp=a%26b"),
			postForm("/examples/gop_file_get_parameters.gop?q=1", url.Values{"key": {"posted"}}),
		}},
		{Name: "post_parameters", Steps: []step{
			get("/examples/gop_file_post_parameters.gop"),
			postForm("/examples/gop_file_post_parameters.gop", url.Values{"key": {"value"}, "other": {"2"}}),
			postForm("/examples/gop_file_post_parameters.gop?key=fromquery", url.Values{"key": {"frombody"}}),
			postForm("/examples/gop_file_post_parameters.gop", url.Values{"key": {"one", "two"}}),
			{Method: "POST", Path: "/examples/gop_file_post_parameters.gop", Multipart: &multipartBody{
				Fields: map[string]string{"key": "multipart"},
			}},
		}},
		{Name: "methods", Steps: []step{
			method("GET", "/examples/gop_file_methods.gop"),
			method("POST", "/examples/gop_file_methods.gop"),
			method("PUT", "/examples/gop_file_methods.gop"),
			method("PATCH", "/examples/gop_file_methods.gop"),
			method("DELETE", "/examples/gop_file_methods.gop"),
			method("OPTIONS", "/examples/gop_file_methods.gop"),
		}},

		// Cookies, sessions, request headers.
		{Name: "cookies", Steps: []step{
			get("/examples/gop_file_cookies.gop"),
			withHeader("/examples/gop_file_cookies.gop", map[string]string{"Cookie": "bob=alice; theme=dark"}),
		}},
		{Name: "sessions", Steps: []step{
			{Path: "/examples/gop_sessions.gop", Extract: []extract{
				{Var: "session", Header: "Set-Cookie", Pattern: `GopSession=([^;\s]*)`},
			}},
			withHeader("/examples/gop_sessions.gop", map[string]string{"Cookie": "GopSession={{session}}"}),
			withHeader("/examples/gop_sessions.gop", map[string]string{
				"Cookie": "GopSession=" + base64.StdEncoding.EncodeToString([]byte(`{"admin":"yes"}`)) + "; other=1",
			}),
			withHeader("/examples/gop_sessions.gop", map[string]string{"Cookie": "GopSession=not-base64!"}),
			withHeader("/__compat/session_destroy.gop", map[string]string{"Cookie": "GopSession={{session}}"}),
		}, Normalize: []rule{re(`(GopSession[=:][\w-]+)\.[\w-]{43}`, "${1}.{{SIGNATURE}}")}},
		{Name: "request_headers", Steps: []step{
			withHeader("/__compat/headers.gop", map[string]string{
				"X-Custom-Header": "custom value",
				"Accept":          "text/html",
				"Cookie":          "a=1; b=2",
			}),
		}},
		{Name: "server_variables", Steps: []step{
			get("/examples/gop_file_server.gop"),
			withHeader("/examples/gop_file_server.gop?x=1&y=two", map[string]string{"Referer": "http://example.test/from"}),
		}},
		{Name: "http_auth", Steps: []step{
			get("/examples/gop_httpauth.gop"),
			withHeader("/examples/gop_httpauth.gop", basicAuth("bob", "wrong")),
			withHeader("/examples/gop_httpauth.gop", basicAuth("bob", "1234")),
		}},

		// Helpers and script-set status and headers.
		{Name: "helpers", Steps: twice("/__compat/helpers.gop")},
		{Name: "status_and_cookie", Steps: twice("/__compat/status.gop")},

		// Aliases: /api/ maps to examples/gop_api.gop.
		{Name: "api_alias", Steps: []step{
			get("/examples/gop_api.gop"),
			get("/api/"),
			get("/api/calculator?a=2&b=40"),
			get("/api/user"),
			method("POST", "/api/user"),
			method("PUT", "/api/user"),
			method("PATCH", "/api/user"),
			method("DELETE", "/api/user"),
			get("/api/admin"),
			get("/api/teapot"),
			get("/api/unknown"),
		}},

		// Uploads: the form field name is FILE_UPLOAD_KEY, read from the form.
		{Name: "upload", Steps: []step{
			{Path: "/examples/gop_file_upload.gop", Extract: []extract{
				{Var: "key", Pattern: `type="file" name="([^"]+)"`},
			}},
			{Method: "POST", Path: "/examples/gop_file_upload.gop", ListDir: "private/uploads", Multipart: &multipartBody{
				Fields:    map[string]string{"key": "with file", "submit": "Upload Image"},
				FileField: "{{key}}",
				FileName:  "hello.txt",
				FileData:  "hello from the compat suite\n",
			}},
		}, Normalize: []rule{re(`upload-\d+`, "upload-{{RANDOM}}")}},

		// Static files and missing paths.
		{Name: "static", Steps: []step{
			get("/examples/text.txt"),
			get("/examples/image.png"),
			get("/favicon.ico"),
			get("/examples/directorylisting/info.txt"),
		}},
		{Name: "not_found", Steps: []step{
			get("/examples/missing.gop"),
			get("/examples/missing.go"),
			get("/examples/missing.txt"),
			get("/missing-dir/"),
		}},

		// Built-in pages.
		{Name: "gop_info", Steps: []step{get("/gop-info/"), get("/gop-info")}},
		{Name: "server_status", Steps: []step{
			get("/server-status/"),
			get("/server-status/?data=true"),
			get("/server-status"),
		}, Normalize: []rule{
			re(`rendered at [^<]*`, "rendered at {{TIME}}"),
			re(`"start":\d+,"count":\d+`, `"start":{{N}},"count":{{N}}`),
		}},
	}
}

// TestCasesCoverExamples fails when a script is added to public/examples
// without a case requesting it, so the suite keeps up with the corpus.
func TestCasesCoverExamples(t *testing.T) {
	examples := filepath.Join(siteRoot(t), "public", "examples")
	requested := map[string]bool{}
	for _, tc := range defaultCases() {
		for _, st := range tc.Steps {
			p := strings.SplitN(st.Path, "?", 2)[0]
			requested[strings.TrimPrefix(p, "/examples/")] = true
		}
	}
	var missing []string
	err := filepath.Walk(examples, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		ext := filepath.Ext(path)
		if info.IsDir() || (ext != ".gop" && ext != ".go") {
			return nil
		}
		rel, _ := filepath.Rel(examples, path)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "dependencies/") || rel == "index.gop" {
			return nil // required by other examples, or served as the directory index
		}
		if !requested[rel] {
			missing = append(missing, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("examples with no compat case: %s", strings.Join(missing, ", "))
	}
}
