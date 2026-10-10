// Package pages renders the server's own pages: errors, /server-status/ and /gop-info/.
// They are html/template files sharing one layout and one logo, embedded in the binary.
package pages

import (
	"bytes"
	"embed"
	"encoding/base64"
	"html/template"
	"io"
	"strings"
	"time"
)

//go:embed templates/*.html
var files embed.FS

//go:embed logo.png
var logo []byte

// pages holds each page parsed together with the layout it fills in
var pages = func() map[string]*template.Template {
	functions := template.FuncMap{
		"logo": func() template.HTMLAttr { // one attribute, so the data URI is written as is
			return template.HTMLAttr(`src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(logo) + `"`)
		},
	}
	parsed := make(map[string]*template.Template)
	for _, name := range []string{"error", "status", "info"} {
		parsed[name] = template.Must(template.New("layout.html").Funcs(functions).
			ParseFS(files, "templates/layout.html", "templates/"+name+".html"))
	}
	return parsed
}()

// Kind is a kind of error page
type Kind int

// The error pages
const (
	NotFound         Kind = iota // details: the URL path
	BadRequest                   // details: what is wrong with the request
	Forbidden                    // details: the URL path
	InternalError                // no details
	ParseError                   // details: the parser's message
	CompileError                 // details: compiler output, explained for .gop scripts
	BuildFailed                  // details: why the go command could not run
	BuildTimeout                 // details: the script
	RunFailed                    // details: the exit status and the script's output
	ExecutionTimeout             // details: the script
)

// errorTexts are each page's title and summary; %s in the summary stands for the details
var errorTexts = map[Kind]struct{ title, summary string }{
	NotFound:         {"Error 404 - Page not found", "The requested URL (%s) was not found."},
	BadRequest:       {"Error 400 - Bad request", "%s"},
	Forbidden:        {"Error 403 - Forbidden", "The requested URL (%s) is not available from your address."},
	InternalError:    {"Error 500 - Internal server error", "The server failed while answering this request."},
	ParseError:       {"There was an error in your .gop source", "%s"},
	CompileError:     {"There was an error in your source", "%s"},
	BuildFailed:      {"The Go compiler could not be run", "%s"},
	BuildTimeout:     {"Build timeout", "Building the script %s took longer than BuildTimeout."},
	RunFailed:        {"The script exited with an error", "%s"},
	ExecutionTimeout: {"Execution timeout", "The script %s ran longer than ExecutionTimeout and was stopped."},
}

// Error renders an error page; signature is the configured ServerSignature
func Error(signature string, kind Kind, details string) string {
	page := errorTexts[kind]
	var b bytes.Buffer
	pages["error"].Execute(&b, map[string]any{ // errors only come from writing, and b does not fail
		"Signature": signature,
		"Title":     page.title,
		"Details":   strings.ReplaceAll(page.summary, "%s", details),
	})
	return b.String()
}

// Setting is one row of /gop-info/
type Setting struct {
	Name, Value string
}

// Info renders /gop-info/
func Info(w io.Writer, signature string, settings []Setting) error {
	return pages["info"].Execute(w, map[string]any{"Signature": signature, "Settings": settings})
}

// Status renders /server-status/; address is where the server listens
func Status(w io.Writer, signature, address string, now time.Time) error {
	return pages["status"].Execute(w, map[string]any{
		"Signature": signature,
		"Address":   address,
		"Now":       now.UTC().Format(time.RFC1123),
	})
}
