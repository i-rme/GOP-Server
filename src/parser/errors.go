package parser

import (
	"errors"
	"fmt"
	"go/token"
	"strconv"
)

// Kinds of parse errors, usable with errors.Is
var (
	// ErrMissingTag means a file does not start with the <!go tag
	ErrMissingTag = errors.New("missing <!go tag")
	// ErrSyntax means a file could not be tokenized or its brackets are unbalanced
	ErrSyntax = errors.New("syntax error")
	// ErrRequire means a require directive is malformed, misplaced or names a missing file
	ErrRequire = errors.New("bad require")
	// ErrRequireCycle means files require each other in a loop
	ErrRequireCycle = errors.New("require cycle")
	// ErrImport means an import declaration is malformed
	ErrImport = errors.New("bad import")
)

// Error is a parse error located in a .gop file
type Error struct {
	File   string // path relative to the requested script's directory
	Line   int    // 1-based, 0 when the error is about the whole file
	Column int    // 1-based, 0 when unknown
	Msg    string
	Err    error // one of the Err* kinds above
}

func (e *Error) Error() string {
	location := e.File
	if e.Line > 0 {
		location += ":" + strconv.Itoa(e.Line)
		if e.Column > 0 {
			location += ":" + strconv.Itoa(e.Column)
		}
	}
	return location + ": " + e.Msg
}

func (e *Error) Unwrap() error {
	return e.Err
}

// errorAt builds an Error pointing at a byte offset of a scanned file
func errorAt(file *token.File, offset int, kind error, format string, args ...interface{}) *Error {
	position := file.Position(file.Pos(offset))
	return &Error{
		File:   position.Filename,
		Line:   position.Line,
		Column: position.Column,
		Msg:    fmt.Sprintf(format, args...),
		Err:    kind,
	}
}
