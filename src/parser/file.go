package parser

import (
	"bytes"
	"go/token"
	"strconv"
)

/*
	file.go turns the statements of one .gop file into the pieces the
	generated program is assembled from:

	- imports, from every top-level import declaration
	- declarations moved to package level: every top-level func (including
	  methods), plus type/var/const declarations written before the last
	  top-level func, so that functions can use them
	- the body that runs inside main(): everything else, in source order,
	  with each require directive marking where a file is inlined
*/

const gopTag = "<!go"

// importSpec is one imported package
type importSpec struct {
	name   string // "", "_", "." or an alias
	path   string
	offset int // where the spec starts in its file
}

// piece is a run of source text, or a require directive when require is set
type piece struct {
	text    string
	offset  int    // where text starts in its file
	require string // path as written in the directive
}

// parsedFile is one .gop file split into its parts
type parsedFile struct {
	file    *token.File
	imports []importSpec
	decls   []piece
	body    []piece
}

// stripTag checks for the <!go tag and blanks it (and a UTF-8 BOM) with spaces,
// so that offsets, lines and columns still match the original file
func stripTag(src []byte) ([]byte, bool) {
	prefix := 0
	if bytes.HasPrefix(src, []byte("\xEF\xBB\xBF")) {
		prefix = 3
	}
	if !bytes.HasPrefix(src[prefix:], []byte(gopTag)) {
		return nil, false
	}
	prefix += len(gopTag)
	if prefix < len(src) && !isSpace(src[prefix]) { // "<!gox" is not the tag
		return nil, false
	}

	stripped := make([]byte, len(src))
	copy(stripped, src)
	for i := 0; i < prefix; i++ {
		stripped[i] = ' '
	}
	return stripped, true
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

// parseFile splits one .gop source into imports, package-level declarations and main body
func parseFile(fset *token.FileSet, name string, src []byte) (*parsedFile, error) {
	src, ok := stripTag(src)
	if !ok {
		return nil, &Error{File: name, Msg: "GOP files and dependencies have to start with the <!go tag", Err: ErrMissingTag}
	}

	file := fset.AddFile(name, -1, len(src))
	lexemes, err := tokenize(file, src)
	if err != nil {
		return nil, err
	}
	statements, err := splitStatements(file, lexemes)
	if err != nil {
		return nil, err
	}

	const (
		inBody = iota
		isImport
		isDecl
	)
	kinds := make([]int, len(statements))
	lastFunc := -1
	for i, s := range statements {
		switch lexemes[s.first].tok {
		case token.IMPORT:
			kinds[i] = isImport
		case token.FUNC:
			if isFuncDecl(lexemes, s) {
				kinds[i] = isDecl
				lastFunc = i
			}
		}
	}
	for i, s := range statements {
		switch lexemes[s.first].tok {
		case token.TYPE, token.VAR, token.CONST:
			if i < lastFunc {
				kinds[i] = isDecl
			}
		}
	}

	parsed := &parsedFile{file: file}
	cursor := 0 // start of the body text not yet emitted
	addText := func(end int) {
		if end > cursor {
			parsed.body = append(parsed.body, piece{text: string(src[cursor:end]), offset: cursor})
		}
	}

	for i, s := range statements {
		switch kinds[i] {
		case isImport:
			specs, err := parseImport(file, lexemes, s)
			if err != nil {
				return nil, err
			}
			parsed.imports = append(parsed.imports, specs...)
			addText(s.start)
			cursor = s.next

		case isDecl:
			if r := findRequires(src, lexemes, s); len(r) > 0 {
				return nil, errorAt(file, r[0].start, ErrRequire, "require is not allowed inside a declaration")
			}
			parsed.decls = append(parsed.decls, piece{text: string(src[s.start:s.end]), offset: s.start})
			addText(s.start)
			cursor = s.next

		default:
			for _, r := range findRequires(src, lexemes, s) {
				if r.path == "" {
					return nil, errorAt(file, r.start, ErrRequire, "require needs a file path")
				}
				addText(r.start)
				parsed.body = append(parsed.body, piece{require: r.path, offset: r.start})
				cursor = r.end
			}
		}
	}
	addText(len(src))

	return parsed, nil
}

// isFuncDecl reports whether a statement starting with func declares a function or method,
// as opposed to calling a function literal such as func() { ... }()
func isFuncDecl(lexemes []lexeme, s statement) bool {
	next := s.first + 1
	if next >= s.last {
		return false
	}
	switch lexemes[next].tok {
	case token.IDENT: // func name(...) or func name[T any](...)
		return true
	case token.LPAREN: // func (receiver) name(...)
		closing := matchingClose(lexemes, next, s.last)
		if closing < 0 || closing+2 >= s.last {
			return false
		}
		afterName := lexemes[closing+2].tok
		return lexemes[closing+1].tok == token.IDENT && (afterName == token.LPAREN || afterName == token.LBRACK)
	}
	return false
}

// parseImport reads the specs of an import declaration
func parseImport(file *token.File, lexemes []lexeme, s statement) ([]importSpec, error) {
	i := s.first + 1
	if i < s.last && lexemes[i].tok == token.LPAREN {
		closing := matchingClose(lexemes, i, s.last)
		if closing != s.last-1 {
			return nil, errorAt(file, lexemes[i].off, ErrImport, "malformed import block")
		}
		var specs []importSpec
		i++
		for i < closing {
			if lexemes[i].tok == token.SEMICOLON {
				i++
				continue
			}
			spec, next, err := parseImportSpec(file, lexemes, i, closing)
			if err != nil {
				return nil, err
			}
			specs = append(specs, spec)
			i = next
			if i < closing && lexemes[i].tok != token.SEMICOLON {
				return nil, errorAt(file, lexemes[i].off, ErrImport, "unexpected %s in import block", describe(lexemes[i]))
			}
		}
		return specs, nil
	}

	spec, next, err := parseImportSpec(file, lexemes, i, s.last)
	if err != nil {
		return nil, err
	}
	if next != s.last {
		return nil, errorAt(file, lexemes[next].off, ErrImport, "unexpected %s after import", describe(lexemes[next]))
	}
	return []importSpec{spec}, nil
}

// parseImportSpec reads `[name] "path"` starting at token i
func parseImportSpec(file *token.File, lexemes []lexeme, i, limit int) (importSpec, int, error) {
	spec := importSpec{offset: lexemes[i].off}
	if i < limit && (lexemes[i].tok == token.IDENT || lexemes[i].tok == token.PERIOD) {
		spec.name = lexemes[i].lit
		if lexemes[i].tok == token.PERIOD {
			spec.name = "."
		}
		i++
	}
	if i >= limit || lexemes[i].tok != token.STRING {
		return spec, i, errorAt(file, spec.offset, ErrImport, "import path must be a string")
	}
	path, err := strconv.Unquote(lexemes[i].lit)
	if err != nil || path == "" {
		return spec, i, errorAt(file, lexemes[i].off, ErrImport, "invalid import path %s", lexemes[i].lit)
	}
	spec.path = path
	return spec, i + 1, nil
}

// requireDirective is a `require "path"` found in a statement
type requireDirective struct {
	path       string
	start, end int // byte range of the directive
}

// findRequires finds `require "path"` directives used as statements anywhere in s,
// including inside blocks such as if or switch bodies
func findRequires(src []byte, lexemes []lexeme, s statement) []requireDirective {
	var found []requireDirective
	for i := s.first; i+1 < s.last; i++ {
		if lexemes[i].tok != token.IDENT || lexemes[i].lit != "require" || lexemes[i+1].tok != token.STRING {
			continue
		}
		if i > s.first {
			switch lexemes[i-1].tok {
			case token.SEMICOLON, token.LBRACE, token.COLON:
			default:
				continue
			}
		}
		if after := i + 2; after < s.last {
			switch lexemes[after].tok {
			case token.SEMICOLON, token.RBRACE:
			default:
				continue
			}
		}
		path, err := strconv.Unquote(lexemes[i+1].lit)
		if err != nil {
			continue
		}
		found = append(found, requireDirective{path: path, start: lexemes[i].off, end: tokenEnd(src, lexemes[i+1])})
	}
	return found
}

// describe names a token for error messages
func describe(l lexeme) string {
	if l.tok == token.SEMICOLON && l.lit == "\n" {
		return "newline"
	}
	if l.lit != "" {
		return strconv.Quote(l.lit)
	}
	return l.tok.String()
}
