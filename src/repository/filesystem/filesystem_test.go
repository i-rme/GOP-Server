package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveInRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "public")
	for _, dir := range []string{"public/sub", "private"} {
		if err := os.MkdirAll(filepath.Join(base, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"public/sub/a.gop", "private/secret.txt"} {
		if err := os.WriteFile(filepath.Join(base, file), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(base, "private"), filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "sub"), filepath.Join(root, "inside")); err != nil {
		t.Fatal(err)
	}
	root, err := RealDir(root)
	if err != nil {
		t.Fatal(err)
	}
	real := func(p string) string {
		r, err := filepath.EvalSymlinks(filepath.Join(base, p))
		if err != nil {
			t.Fatal(err)
		}
		return r
	}

	for urlPath, want := range map[string]string{
		"/sub/a.gop":             real("public/sub/a.gop"),
		"sub/a.gop":              real("public/sub/a.gop"),
		"/inside/a.gop":          real("public/sub/a.gop"),
		"/sub/":                  real("public/sub"),
		"/":                      real("public"),
		"/../private/secret.txt": "", // .. cannot climb above the root
		"/sub/../../private":     "",
		"/escape/secret.txt":     "", // nor can a symlink
		"/missing.gop":           "",
	} {
		got, err := ResolveInRoot(root, urlPath)
		if want == "" {
			if err == nil {
				t.Errorf("%s resolved to %s, want an error", urlPath, got)
			}
			continue
		}
		if err != nil || got != want {
			t.Errorf("%s: got %q, %v; want %q", urlPath, got, err, want)
		}
	}

	if _, err := ResolveInRoot(root, "/escape/secret.txt"); !errors.Is(err, ErrOutsideRoot) {
		t.Errorf("symlink out: %v, want ErrOutsideRoot", err)
	}
}
