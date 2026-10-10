package parser

/*
	glue.go defines the runtime code wrapped around every script
	to turn it into a complete Go program
*/

// runtimeImports are the packages the runtime needs, which scripts can also use without importing them
func runtimeImports(options Options) []importSpec {
	imports := []importSpec{
		{path: "flag"},
		{path: "encoding/json"},
		{path: "html"},
		{path: "strings"},
		{path: "encoding/base64"},
		{path: "strconv"},
		{path: "regexp"},
		{path: "fmt"},
		{path: "time"},
		{path: "os"},
	}
	if options.MySQL {
		imports = append(imports, importSpec{path: "database/sql"}, importSpec{name: "_", path: "github.com/go-sql-driver/mysql"})
	}
	return imports
}

// runtimeVariables defines the globals scripts use
var runtimeVariables = `
// Public Variables
var _GET, _POST, _COOKIE, _SERVER, _SESSION, _HEADER map[string]string

// Private Variables
var __sessionActive = false
var __responseHeader = make([]string, 0)

var _ = os.Hostname	// Dummy assignment to avoid os import not used
`

// runtimeFunctions defines the helpers available to scripts
func runtimeFunctions(options Options) string {
	functions := publicFunctions + privateFunctions(options) + middlewareFunction
	if options.MySQL {
		functions += mysqlFunctions
	}
	return functions
}
