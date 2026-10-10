package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// load writes json into a temp dir that has a public/ directory and loads it from there
func load(t *testing.T, json string) (*Config, error) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "configuration.json")
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)
	return Load(path)
}

func TestShippedConfigurationLoads(t *testing.T) {
	chdir(t, filepath.Join("..", "..", ".."))
	c, err := Load(filepath.Join("config", "configuration.json"))
	if err != nil {
		t.Fatal(err)
	}
	if c.ReadTimeout != Duration(12*time.Second) || c.Aliases["/api/"] != "/examples/gop_api.gop" {
		t.Errorf("unexpected values: %+v", c)
	}
	if len(c.DeniedNetworks) != 1 || c.DeniedNetworks[0].String() != "10.1.2.0/24" {
		t.Errorf("DeniedNetworks = %v", c.DeniedNetworks)
	}
}

func TestEmptyConfigurationUsesDefaults(t *testing.T) {
	c, err := load(t, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	if c.Port != want.Port || c.DocumentRoot != want.DocumentRoot || c.WriteTimeout != want.WriteTimeout {
		t.Errorf("got %+v, want defaults %+v", c, want)
	}
}

func TestDurations(t *testing.T) {
	c, err := load(t, `{"ReadTimeout": 3000000000, "WriteTimeout": "1m30s"}`)
	if err != nil {
		t.Fatal(err)
	}
	if c.ReadTimeout != Duration(3*time.Second) || c.WriteTimeout != Duration(90*time.Second) {
		t.Errorf("ReadTimeout %v, WriteTimeout %v", c.ReadTimeout, c.WriteTimeout)
	}
	if _, err := load(t, `{"ReadTimeout": "soon"}`); err == nil {
		t.Error("an invalid duration loaded")
	}
}

func TestInvalidValues(t *testing.T) {
	for _, tc := range []struct{ json, want string }{
		{`{"DenyFrom": ["10.1.2.0"]}`, `DenyFrom entry "10.1.2.0"`},
		{`{"Port": 0}`, "Port 0"},
		{`{"DocumentRoot": "missing"}`, `DocumentRoot "missing"`},
		{`{"LoggingLevel": "loud"}`, `LoggingLevel "loud"`},
		{`{"SessionCookieName": "a b"}`, "SessionCookieName"},
		{`{"RateLimitEnabled": true, "RateLimitRate": 0}`, "RateLimitRate"},
		{`{"Port": "80"}`, "cannot unmarshal"},
	} {
		_, err := load(t, tc.json)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %v, want it to mention %q", tc.json, err, tc.want)
		}
	}
}

func TestPlaceholderSaltIsReplaced(t *testing.T) {
	a, err := load(t, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := load(t, `{"GopKeySalt": "`+PlaceholderSalt+`"}`)
	if a.GopKeySalt == PlaceholderSalt || a.GopKeySalt == b.GopKeySalt || len(a.Warnings) != 1 {
		t.Errorf("salts %q and %q, warnings %v", a.GopKeySalt, b.GopKeySalt, a.Warnings)
	}
	c, _ := load(t, `{"GopKeySalt": "secret"}`)
	if c.GopKeySalt != "secret" || len(c.Warnings) != 0 {
		t.Errorf("salt %q, warnings %v", c.GopKeySalt, c.Warnings)
	}
}

// chdir changes the working directory for the rest of the test (t.Chdir needs Go 1.24)
func chdir(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(previous) })
}
