package worker

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newWorker(t *testing.T) *Worker {
	t.Helper()
	if testing.Short() {
		t.Skip("builds with the go command")
	}
	w, err := New(Options{CacheRoot: t.TempDir(), BuildTimeout: time.Minute, ExecutionTimeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// build compiles a main function body into a binary
func build(t *testing.T, w *Worker, body string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	binary := filepath.Join(dir, "app")
	files := map[string]string{
		"go.mod":  "module script\n\ngo 1.22\n",
		"main.go": "package main\n\nimport (\"flag\"; \"fmt\"; \"os\"; \"os/exec\")\n\nvar _, _, _ = fmt.Print, os.Exit, exec.Command\n\nfunc main() {\n\tp := flag.String(\"PARAMETERS\", \"\", \"\")\n\tflag.Parse()\n\t_ = p\n" + body + "\n}\n",
	}
	return binary, w.Build(dir, binary, files, "main.go")
}

func TestBuildAndRun(t *testing.T) {
	w := newWorker(t)
	binary, err := build(t, w, `fmt.Print("out:", *p); fmt.Fprint(os.Stderr, "err")`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := w.Run(binary, `["x"]`)
	if err != nil || result.Stdout != `out:["x"]` || result.Stderr != "err" {
		t.Errorf("got %+v, %v", result, err)
	}
}

func TestBuildError(t *testing.T) {
	w := newWorker(t)
	binary, err := build(t, w, `undefinedName()`)
	var buildErr *BuildError
	if !errors.As(err, &buildErr) || !strings.Contains(buildErr.Output, "undefined: undefinedName") || !strings.HasPrefix(buildErr.Output, "exit status 1\n") {
		t.Fatalf("got %v", err)
	}
	if _, err := exec.LookPath(binary); err == nil {
		t.Error("a failed build left a binary")
	}
}

func TestRunFailure(t *testing.T) {
	w := newWorker(t)
	binary, err := build(t, w, `fmt.Print("partial"); os.Exit(3)`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := w.Run(binary, "")
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 3 || result.Stdout != "partial" {
		t.Errorf("got %+v, %v", result, err)
	}
}

func TestRunTimeoutKillsProcessGroup(t *testing.T) {
	w := newWorker(t)
	// The child keeps the output pipe open; the run must still end at the timeout
	binary, err := build(t, w, `c := exec.Command("sleep", "60"); c.Stdout = os.Stdout; c.Start(); for { fmt.Println("busy") }`)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	result, err := w.Run(binary, "")
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("got %v, want ErrTimeout", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("run took %s", elapsed)
	}
	if len(result.Stdout) > maxOutput {
		t.Errorf("kept %d bytes of output, more than %d", len(result.Stdout), maxOutput)
	}
}

func TestRelativeMessages(t *testing.T) {
	w := &Worker{workDir: "/srv/gop"}
	got := w.relative("/srv/gop/cache/abc", "../../public/a.gop:1: x\n/srv/gop/public/b.gop:2: y\n/elsewhere/c.go:3: z")
	want := "public/a.gop:1: x\npublic/b.gop:2: y\n/elsewhere/c.go:3: z"
	if got != want {
		t.Errorf("got %q", got)
	}
}
