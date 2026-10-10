package parser

import (
	"bytes"
	"errors"
	"go/ast"
	goparser "go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var testOptions = Options{HeadersSeparator: "<!--GOP-HEADERS-S3P4R4T0R"}

// scripts lists every .gop file shipped with the server
func scripts(t *testing.T) []string {
	t.Helper()
	var paths []string
	for _, dir := range []string{"../../public", "../../private"} {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && filepath.Ext(path) == ".gop" {
				paths = append(paths, path)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(paths) < 25 {
		t.Fatalf("found only %d scripts", len(paths))
	}
	return paths
}

// TestExamplesMatchLegacy checks that every script the regex-based parser accepted
// still becomes the same program, and that the ones it rejected still fail
func TestExamplesMatchLegacy(t *testing.T) {
	for _, options := range []Options{testOptions, {MySQL: true, HeadersSeparator: "--"}} {
		for _, path := range scripts(t) {
			legacy, legacyValid := legacyProcess(path, options)
			program, err := Process(path, options)

			if !legacyValid {
				if err == nil {
					t.Errorf("%s: legacy parser rejected it, new parser accepted it", path)
				}
				continue
			}
			if err != nil {
				t.Errorf("%s: %v", path, err)
				continue
			}
			if want, got := normalize(t, legacy).String(), normalize(t, program.Source).String(); want != got {
				t.Errorf("%s: programs differ\n--- legacy\n%s\n--- new\n%s", path, want, got)
			}
		}
	}
}

// shape is what a program does, independent of layout and declaration order
type shape struct {
	imports []string // sorted
	decls   []string // package-level declarations other than main, sorted
	body    []string // statements of main, in order
}

func (s shape) String() string {
	return strings.Join(s.imports, "\n") + "\n\n" + strings.Join(s.decls, "\n") + "\n\nfunc main() {\n" + strings.Join(s.body, "\n") + "\n}"
}

// normalize parses a program into its shape
func normalize(t *testing.T, src string) shape {
	t.Helper()
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf("generated source does not parse: %v\n%s", err, src)
	}

	print := func(node interface{}) string {
		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, node); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}

	var s shape
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.IMPORT {
				s.decls = append(s.decls, print(d))
				continue
			}
			for _, spec := range d.Specs {
				s.imports = append(s.imports, print(spec))
			}
		case *ast.FuncDecl:
			if d.Name.Name != "main" || d.Recv != nil {
				s.decls = append(s.decls, print(d))
				continue
			}
			for _, stmt := range d.Body.List {
				s.body = append(s.body, print(stmt))
			}
		}
	}
	sort.Strings(s.imports)
	sort.Strings(s.decls)
	return s
}

// TestExamplesBuild compiles every script with the Go toolchain
func TestExamplesBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("builds with the go command")
	}
	failing := map[string]string{
		// Kept in the examples on purpose: the error page shows this line of the script
		"gop_file_syntax_error.gop": "gop_file_syntax_error.gop:9:17: invalid operation",
		// Only meant to be required: on its own its variables are unused
		"template.gop": "template.gop:3:5: declared and not used: TemplateStyles",
	}

	dir := t.TempDir()
	for _, path := range scripts(t) {
		name := filepath.Base(path)
		program, err := Process(path, testOptions)
		if err != nil {
			continue // covered by TestExamplesMatchLegacy
		}
		if bytes.Contains([]byte(program.Source), []byte("_mysqli_")) {
			continue // needs MySQL support and its driver
		}

		source := filepath.Join(dir, strings.TrimSuffix(name, ".gop")+".go")
		if err := os.WriteFile(source, []byte(program.Source), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("go", "build", "-o", os.DevNull, source)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()

		if want, ok := failing[name]; ok {
			if err == nil || !strings.Contains(string(output), want) {
				t.Errorf("%s: want a build error mentioning %q, got:\n%s", name, want, output)
			}
			continue
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Errorf("%s does not build:\n%s", name, output)
		} else if err != nil {
			t.Fatal(err)
		}
	}
}

// TestRuntimeWithMySQLBuilds compiles a script with the MySQL helpers, inside this module for the driver
func TestRuntimeWithMySQLBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("builds with the go command")
	}
	program, err := Process("../../public/examples/gop_mysql.gop", Options{MySQL: true, HeadersSeparator: "--"})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(source, []byte(program.Source), 0o644); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("go", "build", "-o", os.DevNull, source).CombinedOutput(); err != nil {
		t.Errorf("does not build: %v\n%s", err, output)
	}
}
