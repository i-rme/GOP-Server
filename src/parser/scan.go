package parser

import (
	"bytes"
	"go/scanner"
	"go/token"
)

/*
	scan.go splits a .gop file into Go tokens and top-level statements.

	Tokenizing with go/scanner means string literals ("...", `...`, '...')
	and comments are understood exactly as the Go compiler understands
	them, so keywords inside them are never mistaken for code.
*/

// lexeme is one Go token with its byte offset in the source
type lexeme struct {
	tok token.Token
	lit string
	off int
}

// statement is a run of tokens ending at a semicolon outside any brackets
type statement struct {
	first, last int // token index range [first, last)
	start, end  int // byte range of the statement, without its terminator
	next        int // byte offset just after the terminator (after an explicit ';')
}

// tokenize scans src as Go source and returns its tokens, ending with EOF.
// It fails on the first lexical error, such as an unterminated string or comment.
func tokenize(file *token.File, src []byte) ([]lexeme, error) {
	var firstErr error
	onError := func(pos token.Position, msg string) {
		if firstErr == nil {
			firstErr = &Error{File: pos.Filename, Line: pos.Line, Column: pos.Column, Msg: msg, Err: ErrSyntax}
		}
	}

	var s scanner.Scanner
	s.Init(file, src, onError, 0) // comments are skipped

	var lexemes []lexeme
	for {
		pos, tok, lit := s.Scan()
		lexemes = append(lexemes, lexeme{tok: tok, lit: lit, off: file.Offset(pos)})
		if tok == token.EOF {
			break
		}
	}

	if firstErr != nil {
		return nil, firstErr
	}
	return lexemes, nil
}

// splitStatements groups tokens into top-level statements and checks that brackets are balanced
func splitStatements(file *token.File, lexemes []lexeme) ([]statement, error) {
	var statements []statement
	var open []int // indexes of unclosed ( [ {
	inHeader := false
	current := -1

	closeStatement := func(last, end, next int) {
		statements = append(statements, statement{
			first: current,
			last:  last,
			start: lexemes[current].off,
			end:   end,
			next:  next,
		})
		current = -1
	}

	for i, l := range lexemes {
		if l.tok == token.EOF {
			if len(open) > 0 {
				opening := lexemes[open[len(open)-1]]
				return nil, errorAt(file, opening.off, ErrSyntax, "unclosed %s", opening.tok)
			}
			if current >= 0 {
				closeStatement(i, l.off, l.off)
			}
			break
		}

		// Semicolons inside "for a; b; c {" or "if x := f(); x {" headers do not end the statement
		explicit := l.lit == ";"
		if l.tok == token.SEMICOLON && len(open) == 0 && !(inHeader && explicit) {
			if current >= 0 {
				next := l.off
				if explicit {
					next++
				}
				closeStatement(i, l.off, next)
			}
			continue
		}

		if current < 0 {
			current = i
		}

		switch l.tok {
		case token.FOR, token.IF, token.SWITCH:
			if len(open) == 0 {
				inHeader = true
			}
		case token.LPAREN, token.LBRACK, token.LBRACE:
			if l.tok == token.LBRACE && len(open) == 0 {
				inHeader = false
			}
			open = append(open, i)
		case token.RPAREN, token.RBRACK, token.RBRACE:
			if len(open) == 0 || !closes(lexemes[open[len(open)-1]].tok, l.tok) {
				return nil, errorAt(file, l.off, ErrSyntax, "unexpected %s", l.tok)
			}
			open = open[:len(open)-1]
		}
	}

	return statements, nil
}

func closes(opening, closing token.Token) bool {
	switch opening {
	case token.LPAREN:
		return closing == token.RPAREN
	case token.LBRACK:
		return closing == token.RBRACK
	case token.LBRACE:
		return closing == token.RBRACE
	}
	return false
}

// matchingClose returns the index of the bracket closing the one at index i, or -1
func matchingClose(lexemes []lexeme, i, limit int) int {
	depth := 0
	for j := i; j < limit; j++ {
		switch lexemes[j].tok {
		case token.LPAREN, token.LBRACK, token.LBRACE:
			depth++
		case token.RPAREN, token.RBRACK, token.RBRACE:
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

// tokenEnd returns the byte offset just past a token
func tokenEnd(src []byte, l lexeme) int {
	if l.tok == token.STRING && len(l.lit) > 0 && l.lit[0] == '`' {
		// The scanner drops '\r' from raw strings, so len(lit) can be short
		closing := bytes.IndexByte(src[l.off+1:], '`')
		return l.off + 1 + closing + 1
	}
	if l.lit != "" && l.tok != token.SEMICOLON {
		return l.off + len(l.lit)
	}
	return l.off + len(l.tok.String())
}
