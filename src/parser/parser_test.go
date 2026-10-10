package parser

import (
	"errors"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// translate writes files into a temporary directory and processes main.gop
func translate(t *testing.T, files map[string]string) (*Program, error) {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Process(filepath.Join(dir, "main.gop"), testOptions)
}

// mustTranslate processes main.gop and returns what the script added to the runtime,
// as text: imports, package-level declarations and main body, separated by blank lines
func mustTranslate(t *testing.T, files map[string]string) (imports, decls, body string) {
	t.Helper()
	program, err := translate(t, files)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	runtime := normalize(t, "package main\n"+runtimeConstants(testOptions)+"func main() { __housekeeping() }\n"+runtimeCode(testOptions))
	got := normalize(t, program.Source)
	return strings.Join(without(got.imports, importLines(runtimeImports(testOptions))), "\n"),
		strings.Join(without(got.decls, runtime.decls), "\n"),
		strings.Join(without(got.body, runtime.body), "\n")
}

func importLines(specs []importSpec) []string {
	var lines []string
	for _, spec := range specs {
		lines = append(lines, importLine(spec))
	}
	return lines
}

// without removes one occurrence of each of drop from list
func without(list, drop []string) []string {
	var kept []string
	pending := append([]string(nil), drop...)
next:
	for _, item := range list {
		for i, d := range pending {
			if item == d {
				pending = append(pending[:i], pending[i+1:]...)
				continue next
			}
		}
		kept = append(kept, item)
	}
	return kept
}

func TestTokensInsideStringsAndComments(t *testing.T) {
	_, decls, body := mustTranslate(t, map[string]string{"main.gop": `<!go
// func helper() { require "missing.gop" }
/* import "net/http"
   func other() {} */
fmt.Println(" func foo(x) {", "}")
fmt.Println("require \"missing.gop\"")
fmt.Println(` + "`" + `
import "os"
func raw() {
` + "`" + `)
fmt.Println('{')
fmt.Println("<!go ")
`})
	if decls != "" {
		t.Errorf("nothing should be hoisted, got:\n%s", decls)
	}
	for _, want := range []string{`" func foo(x) {"`, `"require \"missing.gop\""`, "\nimport \"os\"\nfunc raw() {\n", `'{'`, `"<!go "`} {
		if !strings.Contains(body, want) {
			t.Errorf("main body lost %s:\n%s", want, body)
		}
	}
}

func TestFunctionsAreHoisted(t *testing.T) {
	_, decls, body := mustTranslate(t, map[string]string{"main.gop": `<!go
type counter struct{ n int }

var greeting = "hi"

func (c *counter) Add() { c.n++ }

func braces() string {
	return "}{" + ` + "`}`" + `
}

func generic[T any](v T) T { return v }

c := &counter{}
c.Add()
func() { fmt.Println("closure runs in main") }()
later := func() {}
later()
var local = 1
fmt.Println(braces(), generic(local), greeting)
`})
	for _, want := range []string{"type counter struct", `var greeting = "hi"`, "func (c *counter) Add()", "func braces() string", "func generic[T any]"} {
		if !strings.Contains(decls, want) {
			t.Errorf("%q should be at package level:\n%s", want, decls)
		}
	}
	for _, want := range []string{"c.Add()", "closure runs in main", "later := func()", "var local = 1"} {
		if !strings.Contains(body, want) {
			t.Errorf("%q should stay in main:\n%s", want, body)
		}
	}
}

func TestImports(t *testing.T) {
	imports, _, _ := mustTranslate(t, map[string]string{
		"main.gop": `<!go
import "math/rand"
fmt.Println(rand.Intn(1), sort.IsSorted(nil))
import (
	"sort"
	"fmt"
	str "strings"; _ "embed"
)
import . "math"
require "lib.gop"
fmt.Println(str.ToUpper("x"), Pi)
`,
		"lib.gop": `<!go
import "math/rand"
import "sort"
`,
	})
	// "fmt" is already imported by the runtime, the others once each
	want := strings.Join([]string{`"math/rand"`, `"sort"`, `. "math"`, `_ "embed"`, `str "strings"`}, "\n")
	if imports != want {
		t.Errorf("imports:\n%s\nwant:\n%s", imports, want)
	}
}

func TestRequire(t *testing.T) {
	program, err := translate(t, map[string]string{
		"main.gop": `<!go
fmt.Println("before")
require "lib/a.gop"
if true {
	require "lib/b.gop"
}
fmt.Println("x"); require "lib/c.gop"; fmt.Println("after")
`,
		"lib/a.gop": "<!go\r\nrequire \"c.gop\"\r\nfmt.Println(\"a\")\r\n",
		"lib/b.gop": "<!go fmt.Println(\"b\")",
		"lib/c.gop": "<!go\nfunc helper() string { return \"c\" }\nfmt.Println(helper())\n",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Each file's body is inlined where it is required, its functions declared once
	source := program.Source
	rest := source[strings.Index(source, "func main()"):]
	for _, want := range []string{`"before"`, `helper()`, `"a"`, `"b"`, `"x"`, `helper()`, `"after"`} {
		i := strings.Index(rest, want)
		if i < 0 {
			t.Fatalf("main body is missing %s or has it out of order:\n%s", want, source)
		}
		rest = rest[i+len(want):]
	}
	if strings.Count(source, "func helper()") != 1 {
		t.Errorf("helper should be declared once:\n%s", source)
	}

	var names []string
	for _, file := range program.Files {
		names = append(names, filepath.Base(file))
	}
	if got := strings.Join(names, " "); got != "main.gop a.gop c.gop b.gop" {
		t.Errorf("Files = %s", got)
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		kind  error
		msg   string
	}{
		{"empty file", map[string]string{"main.gop": ""}, ErrMissingTag, "main.gop: GOP files"},
		{"no tag", map[string]string{"main.gop": `fmt.Println("<!go")`}, ErrMissingTag, "main.gop: "},
		{"tag without space", map[string]string{"main.gop": `<!gofmt.Println()`}, ErrMissingTag, "main.gop: "},
		{"dependency without tag", map[string]string{"main.gop": "<!go\nrequire \"a.gop\"", "a.gop": "x := 1"}, ErrMissingTag, "a.gop: "},
		{"missing dependency", map[string]string{"main.gop": "<!go\n\nrequire \"nope.gop\""}, ErrRequire, "main.gop:3:1: file not found: nope.gop"},
		{"self require", map[string]string{"main.gop": "<!go\nrequire \"main.gop\""}, ErrRequireCycle, "main.gop:2:1: require cycle: main.gop -> main.gop"},
		{"require cycle", map[string]string{
			"main.gop":  "<!go\nrequire \"dir/a.gop\"",
			"dir/a.gop": "<!go\nrequire \"b.gop\"",
			"dir/b.gop": "<!go\nfmt.Println()\nrequire \"../main.gop\"",
		}, ErrRequireCycle, "dir/b.gop:3:1: require cycle: main.gop -> dir/a.gop -> dir/b.gop -> main.gop"},
		{"require in a function", map[string]string{"main.gop": "<!go\nfunc f() {\n\trequire \"a.gop\"\n}"}, ErrRequire, "main.gop:3:2: require is not allowed inside a declaration"},
		{"unterminated string", map[string]string{"main.gop": "<!go\nfmt.Println(\"oops)\n"}, ErrSyntax, "main.gop:2:13: string literal not terminated"},
		{"unterminated comment", map[string]string{"main.gop": "<!go\n/* forever"}, ErrSyntax, "main.gop:2:1: comment not terminated"},
		{"unclosed brace", map[string]string{"main.gop": "<!go\nif true {\n"}, ErrSyntax, "main.gop:2:9: unclosed {"},
		{"extra brace", map[string]string{"main.gop": "<!go\n}\n"}, ErrSyntax, "main.gop:2:1: unexpected }"},
		{"bad import", map[string]string{"main.gop": "<!go\nimport fmt\n"}, ErrImport, "main.gop:2:8: import path must be a string"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := translate(t, test.files)
			if !errors.Is(err, test.kind) {
				t.Fatalf("got %v, want a %v", err, test.kind)
			}
			if !strings.HasPrefix(err.Error(), test.msg) {
				t.Errorf("got %q, want it to start with %q", err, test.msg)
			}
		})
	}
}

func TestLineDirectivesPointAtScript(t *testing.T) {
	program, err := translate(t, map[string]string{"main.gop": "<!go\nimport \"os\"\n\nfunc f() {}\nfmt.Println(os.Args)\n"})
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "", program.Source, 0)
	if err != nil {
		t.Fatal(err)
	}

	locations := map[string]string{}
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.FuncDecl:
			locations[n.Name.Name] = fset.Position(n.Pos()).String()
		case *ast.CallExpr:
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Println" && locations["Println"] == "" {
				locations["Println"] = fset.Position(n.Pos()).String()
			}
		}
		return true
	})
	for name, want := range map[string]string{"f": "main.gop:4:1", "Println": "main.gop:5:1", "main": generatedFile + ":", "_echo": "runtime.go:"} {
		if !strings.Contains(locations[name], want) {
			t.Errorf("%s is reported at %s, want %s", name, locations[name], want)
		}
	}
}

func TestSplitOutput(t *testing.T) {
	sep := testOptions.HeadersSeparator
	trailer := sep + ` {"status":404,"headers":[["X","1"]]}` + "\n"
	tests := []struct {
		in, output string
		trailer    *Trailer
	}{
		{"hello\n" + trailer, "hello\n", &Trailer{404, [][2]string{{"X", "1"}}}},
		{"exited early", "exited early", nil},
		{"fake " + sep + " {}\n" + trailer, "fake " + sep + " {}\n", &Trailer{404, [][2]string{{"X", "1"}}}},
		{"broken\n" + sep + " {", "broken\n" + sep + " {", nil},
	}
	for _, test := range tests {
		output, trailer := SplitOutput(test.in, sep)
		if output != test.output || !reflect.DeepEqual(trailer, test.trailer) {
			t.Errorf("SplitOutput(%q) = %q, %+v", test.in, output, trailer)
		}
	}
}
