package parser

import "path/filepath"

/*
	module.go gives every program the module it is built in. The MySQL
	runtime needs the driver, pinned here with its checksums (the same as in
	the server's go.sum) so a build needs no go.mod editing and no network
	when the module cache already has it.
*/

const goMod = "module script\n\ngo 1.22\n"

const (
	mysqlRequire = "\nrequire github.com/go-sql-driver/mysql v1.5.0\n"
	mysqlSum     = "github.com/go-sql-driver/mysql v1.5.0 h1:ozyZYNQW3x3HtqT1jira07DN2PArx2v7/mN66gGcHOs=\n" +
		"github.com/go-sql-driver/mysql v1.5.0/go.mod h1:DCzpHaOWr8IXmIStZouvnhqoel9Qv2LBy8hT2VhHyBg=\n"
)

// Plain is a standalone .go program, built from where it is so compiler messages name it
func Plain(path string, src []byte) *Program {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return &Program{Source: string(src), Files: []string{path}, plain: true}
}

// Module returns the files to write into an empty build directory, by name,
// and the file to compile there
func (p *Program) Module() (files map[string]string, main string) {
	switch {
	case p.plain:
		return map[string]string{"go.mod": goMod}, p.Files[0]
	case p.mysql:
		return map[string]string{"main.go": p.Source, "go.mod": goMod + mysqlRequire, "go.sum": mysqlSum}, "main.go"
	default:
		return map[string]string{"main.go": p.Source, "go.mod": goMod}, "main.go"
	}
}
