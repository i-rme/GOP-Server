package parser

import (
	"errors"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Options are the server settings baked into the generated program
type Options struct {
	MySQL             bool   // adds the _mysqli_* helpers and the MySQL driver
	SessionCookieName string // cookie the session is stored in
	HeadersSeparator  string // line prefix separating the output from the headers
}

// Program is the Go translation of a .gop script
type Program struct {
	Source string   // a complete `package main` program
	Files  []string // absolute paths of every .gop file read, the script first
}

// runtimeFile names the generated code in compiler messages
const runtimeFile = "gop-runtime.go"

// Process reads a .gop script, inlines the files it requires and returns it as a Go program
func Process(scriptPath string, options Options) (*Program, error) {
	return process(scriptPath, options, os.ReadFile)
}

func process(scriptPath string, options Options, readFile func(string) ([]byte, error)) (*Program, error) {
	root, err := filepath.Abs(scriptPath)
	if err != nil {
		return nil, err
	}

	t := &translation{
		readFile: readFile,
		fset:     token.NewFileSet(),
		baseDir:  filepath.Dir(root),
		bodies:   make(map[string]string),
		imports:  make(map[importSpec]bool),
	}
	for _, spec := range runtimeImports(options) {
		t.imports[spec] = true
	}

	body, err := t.include(root, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Program{Source: t.assemble(options, body), Files: t.files}, nil
}

// translation holds the state of one Process call
type translation struct {
	readFile func(string) ([]byte, error)
	fset     *token.FileSet
	baseDir  string // paths in errors are shown relative to the script's directory

	files   []string          // every file read, in order
	bodies  map[string]string // translated main() body of each file read
	imports map[importSpec]bool
	extra   strings.Builder // user imports not already provided by the runtime
	decls   strings.Builder // user package-level declarations
}

// include translates one file and returns its main() body.
// chain holds the files currently being required, to detect cycles;
// site is the require directive that asked for the file, for error messages.
func (t *translation) include(path string, chain []string, site *Error) (string, error) {
	for i, p := range chain {
		if p == path {
			var names []string
			for _, q := range chain[i:] {
				names = append(names, t.display(q))
			}
			names = append(names, t.display(path))
			return "", t.at(site, path, ErrRequireCycle, "require cycle: "+strings.Join(names, " -> "))
		}
	}

	// A file required twice is inlined twice, but its imports and declarations only once
	if body, done := t.bodies[path]; done {
		return body, nil
	}

	src, err := t.readFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", t.at(site, path, ErrRequire, "file not found: "+t.display(path))
		}
		return "", t.at(site, path, ErrRequire, err.Error())
	}

	parsed, err := parseFile(t.fset, t.display(path), src)
	if err != nil {
		return "", err
	}
	t.files = append(t.files, path)

	for _, spec := range parsed.imports {
		key := importSpec{name: spec.name, path: spec.path}
		if !t.imports[key] {
			t.imports[key] = true
			t.extra.WriteString("import " + lineDirective(path, parsed.file, spec.offset) + importLine(key) + "\n")
		}
	}
	for _, decl := range parsed.decls {
		t.decls.WriteString("\n" + lineDirective(path, parsed.file, decl.offset) + decl.text + "\n")
	}

	chain = append(chain, path)
	var body strings.Builder
	for _, p := range parsed.body {
		if p.require == "" {
			body.WriteString(lineDirective(path, parsed.file, p.offset) + p.text)
			continue
		}
		position := parsed.file.Position(parsed.file.Pos(p.offset))
		site := &Error{File: position.Filename, Line: position.Line, Column: position.Column}
		required, err := t.include(filepath.Join(filepath.Dir(path), p.require), chain, site)
		if err != nil {
			return "", err
		}
		body.WriteString("\n" + required + "\n")
	}

	t.bodies[path] = body.String()
	return body.String(), nil
}

// assemble wraps the translated body and declarations with the runtime
func (t *translation) assemble(options Options, body string) string {
	var out strings.Builder
	out.WriteString("package main\n\n")
	for _, spec := range runtimeImports(options) {
		out.WriteString("import " + importLine(spec) + "\n")
	}
	out.WriteString(t.extra.String())

	resetLines(&out)
	out.WriteString(runtimeVariables)
	out.WriteString("\nfunc main() {\n")
	out.WriteString(body)
	out.WriteString("\n")
	resetLines(&out)
	out.WriteString("__housekeeping()\n}\n")
	out.WriteString(runtimeFunctions(options))
	out.WriteString(t.decls.String())

	return out.String()
}

// display returns a path relative to the script's directory, for messages
func (t *translation) display(path string) string {
	if rel, err := filepath.Rel(t.baseDir, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return path
}

// at locates an error at the require directive that included path, or at path itself for the script
func (t *translation) at(site *Error, path string, kind error, msg string) *Error {
	if site == nil {
		return &Error{File: t.display(path), Msg: msg, Err: kind}
	}
	return &Error{File: site.File, Line: site.Line, Column: site.Column, Msg: msg, Err: kind}
}

func importLine(spec importSpec) string {
	if spec.name == "" {
		return strconv.Quote(spec.path)
	}
	return spec.name + " " + strconv.Quote(spec.path)
}

// lineDirective makes the compiler report what follows at its place in the .gop file,
// so build errors point at the line the author wrote
func lineDirective(path string, file *token.File, offset int) string {
	position := file.Position(file.Pos(offset))
	return "/*line " + path + ":" + strconv.Itoa(position.Line) + ":" + strconv.Itoa(position.Column) + "*/"
}

// resetLines ends the previous line directive: what follows is reported as
// gop-runtime.go with its real line number in the generated source
func resetLines(out *strings.Builder) {
	if !strings.HasSuffix(out.String(), "\n") {
		out.WriteString("\n")
	}
	line := strings.Count(out.String(), "\n") + 2 // the directive takes one line
	out.WriteString("//line " + runtimeFile + ":" + strconv.Itoa(line) + "\n")
}
