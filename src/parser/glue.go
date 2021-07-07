package parser

/*
	Glue.go defines variables used by parser.AddContext
	to form a valid go file from gop source adding middleware
	to provide extra functionality
*/

// PackageDefinition Required to start a valid go file
var PackageDefinition = "package main\r\n"

// ImportsDefinition Required to receive and parse GET parameters
var ImportsDefinition = `
import "flag"
import "encoding/json"
import "html"
import "strings"
import "encoding/base64"
import "strconv"
import "regexp"
import "fmt"
import "time"
import "os"
`

// ImportsDefinitionMySQL Required to support MySQL queries
var ImportsDefinitionMySQL = `
import "database/sql"
import _ "github.com/go-sql-driver/mysql"
`

// VariablesDefinition defines variables
var VariablesDefinition = `
// Public Variables
var _GET, _POST, _COOKIE, _SERVER, _SESSION, _HEADER map[string]string

// Private Variables
var __sessionActive = false
var __responseHeader = make([]string, 0)

var _ = os.Hostname	// Dummy assignment to avoid os import not used
`

// MainFunction Required to wrap the code to be run
var MainFunction = "\r\nfunc main() {"

// MainFunctionEnd Required to wrap the code to be run
// Headers required to allow redirections and custom headers
var MainFunctionEnd = "\r\n" +
	"__housekeeping()" + "\r\n" +
	"\r\n}\r\n"

// FunctionsDefinition defines custom functions
var FunctionsDefinition = publicFunctions + privateFunctions + middlewareFunction

// FunctionsDefinitionMySQL defines custom functions
var FunctionsDefinitionMySQL = mysqlFunctions
