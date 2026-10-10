package parser

import (
	_ "embed"
	goparser "go/parser"
	"go/token"
	"strconv"
	"strings"
)

/*
	glue.go loads the runtime wrapped around every script. The runtime is
	plain Go in runtime.go and runtime_mysql.go, embedded here and kept out
	of the server build by their build constraint.
*/

//go:embed runtime.go
var runtimeSource string

//go:embed runtime_mysql.go
var runtimeMySQLSource string

// runtimeFile is an embedded runtime file split into its imports and the code after them
type runtimeFile struct {
	name    string
	imports []importSpec
	code    string // starts at the beginning of a line
	line    int    // line of code in the file
}

var runtimeFiles = [...]runtimeFile{
	splitRuntime("runtime.go", runtimeSource),
	splitRuntime("runtime_mysql.go", runtimeMySQLSource),
}

func splitRuntime(name, src string) runtimeFile {
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, name, src, goparser.ImportsOnly)
	if err != nil {
		panic(err) // the embedded runtime is part of the build; its tests compile it
	}

	runtime := runtimeFile{name: name}
	for _, spec := range file.Imports {
		path, _ := strconv.Unquote(spec.Path.Value)
		imported := importSpec{path: path}
		if spec.Name != nil {
			imported.name = spec.Name.Name
		}
		runtime.imports = append(runtime.imports, imported)
	}

	end := fset.Position(file.Decls[len(file.Decls)-1].End()) // only imports are parsed
	start := end.Offset + strings.IndexByte(src[end.Offset:], '\n') + 1
	runtime.code = src[start:]
	runtime.line = end.Line + 1
	return runtime
}

// runtimeFor returns the runtime files a script needs
func runtimeFor(options Options) []runtimeFile {
	if options.MySQL {
		return runtimeFiles[:]
	}
	return runtimeFiles[:1]
}

// runtimeImports are the packages the runtime imports, which scripts can also use without importing them
func runtimeImports(options Options) []importSpec {
	var imports []importSpec
	seen := make(map[importSpec]bool)
	for _, runtime := range runtimeFor(options) {
		for _, spec := range runtime.imports {
			if !seen[spec] {
				seen[spec] = true
				imports = append(imports, spec)
			}
		}
	}
	return imports
}

// runtimeConstants are the server settings the runtime reads
func runtimeConstants(options Options) string {
	return "const __headersSeparator = " + strconv.Quote(options.HeadersSeparator) + "\n"
}
