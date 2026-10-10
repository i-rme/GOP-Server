package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestErrorEscapesDetails(t *testing.T) {
	page := Error("Sig <1>", NotFound, "<script>x</script>")
	for _, want := range []string{
		"<title>Error 404 - Page not found - Sig &lt;1&gt;</title>",
		"The requested URL (&lt;script&gt;x&lt;/script&gt;) was not found.",
		`<img src="data:image/png;base64,iVBOR`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q:\n%s", want, page)
		}
	}
}

func TestEveryKindHasText(t *testing.T) {
	for kind := NotFound; kind <= ExecutionTimeout; kind++ {
		if errorTexts[kind].title == "" {
			t.Errorf("kind %d has no title", kind)
		}
	}
}

func TestStatusAndInfoRender(t *testing.T) {
	var b strings.Builder
	if err := Status(&b, "Sig", "127.0.0.1:80", time.Unix(0, 0)); err != nil || !strings.Contains(b.String(), "Status for 127.0.0.1:80") {
		t.Errorf("status: %v\n%s", err, b.String())
	}
	b.Reset()
	if err := Info(&b, "Sig", []Setting{{"Port", "80"}}); err != nil || !strings.Contains(b.String(), "<th>Port</th><td>80</td>") {
		t.Errorf("info: %v\n%s", err, b.String())
	}
}

func TestExplain(t *testing.T) {
	script := filepath.Join(t.TempDir(), "a.gop")
	if err := os.WriteFile(script, []byte("<!go\nx := 1\ny := \"a\" + x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := "exit status 1\n# command-line-arguments\n" + script + ":3:6: invalid operation\n"
	got := Explain(output)
	for _, want := range []string{"error on line 3.", "invalid operation", "1.   <!go", "3.   y := \"a\" + x"} {
		if !strings.Contains(got, want) {
			t.Errorf("explanation lacks %q:\n%s", want, got)
		}
	}
	if Explain("go: not found") != "go: not found" {
		t.Error("unreadable output was changed")
	}
}
