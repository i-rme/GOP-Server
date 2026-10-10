package controller

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/i-rme/gop-server/src/parser"
)

func TestProgramsReparseOnlyChangedScripts(t *testing.T) {
	dir := t.TempDir()
	script, required := filepath.Join(dir, "a.gop"), filepath.Join(dir, "b.gop")
	write := func(path, content string, age time.Duration) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-age)
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}
	write(script, "a", time.Hour)
	write(required, "b", time.Hour)

	var p programs
	parses := 0
	parse := func() (*parser.Program, error) {
		parses++
		return &parser.Program{Files: []string{script, required}}, nil
	}
	get := func(want int, why string) {
		t.Helper()
		if _, err := p.get(script, parse); err != nil {
			t.Fatal(err)
		}
		if parses != want {
			t.Errorf("%s: %d parses, want %d", why, parses, want)
		}
	}

	get(1, "first request")
	get(1, "unchanged files")
	write(required, "bb", time.Hour)
	get(2, "required file grew")
	write(required, "cc", 2*time.Hour)
	get(3, "required file touched")
	write(script, "a", 0)
	get(4, "fresh edit")
	get(5, "fresh files are not trusted yet")
	if err := os.Remove(required); err != nil {
		t.Fatal(err)
	}
	get(6, "required file removed")
}
