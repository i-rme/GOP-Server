// Package config loads and validates config/configuration.json.
package config

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// PlaceholderSalt is the GopKeySalt shipped in the example configuration
const PlaceholderSalt = "INSERT_RANDOM_SECURE_STRING_HERE"

// Config is the server configuration. Every key is optional; see Default for the
// values of missing keys. Relative paths are resolved against the working directory.
type Config struct {
	Port           int      // port the server listens on
	Address        string   // address the server listens on, empty for all
	ReadTimeout    Duration // from accepting the connection to reading the whole request
	WriteTimeout   Duration // from reading the request headers to writing the whole response
	IdleTimeout    Duration // keep-alive timeout
	MaxHeaderBytes int      // largest request header accepted

	DocumentRoot string // directory served, holding the .gop, .go and static files
	CacheRoot    string // directory for compiled scripts, emptied at startup
	LogRoot      string // directory for log files, one per start
	UploadsRoot  string // directory uploaded files are saved to

	Aliases  map[string]string // URL prefix → script, relative to DocumentRoot
	DenyFrom []string          // CIDR ranges answered with IPDeniedScript

	StatusAllowFrom []string // CIDR ranges allowed to see /server-status/ and /gop-info/

	ServerSignature        string // Server header and page footer
	DirectoryListingScript string // run for directories without index.gop
	IPDeniedScript         string // run for clients in DenyFrom
	SessionCookieName      string // cookie holding the script session
	GopHeadersSeparator    string // line separating a script's output from its headers
	LoggingLevel           string // debug, basic or error
	GopKeySalt             string // secret mixed into server-generated keys

	RateLimitEnabled  bool   // answer clients over the limit with RateLimitedScript
	RateLimitRate     int    // requests allowed per client and period
	RateLimitPeriod   int    // period length in seconds
	RateLimitedScript string // run for clients over the limit

	MySQLSupportEnabled bool     // add the _mysqli_* helpers to scripts
	RunScriptsAsNobody  bool     // build and run scripts as the unprivileged nobody user
	BuildTimeout        Duration // limit for compiling a script
	ExecutionTimeout    Duration // limit for one run of a script

	DeniedNetworks []*net.IPNet `json:"-"` // DenyFrom, parsed
	StatusNetworks []*net.IPNet `json:"-"` // StatusAllowFrom, parsed
	Warnings       []string     `json:"-"` // problems that do not stop the server, for the log
}

// Duration is a time.Duration written in JSON as nanoseconds (12000000000) or as a string ("12s")
type Duration time.Duration

// UnmarshalJSON accepts both forms of a Duration
func (d *Duration) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		parsed, err := time.ParseDuration(s)
		*d = Duration(parsed)
		return err
	}
	var ns int64
	if err := json.Unmarshal(b, &ns); err != nil {
		return fmt.Errorf("duration must be nanoseconds or a string like \"12s\": %s", b)
	}
	*d = Duration(ns)
	return nil
}

// Std returns d as a time.Duration
func (d Duration) Std() time.Duration {
	return time.Duration(d)
}

func (d Duration) String() string {
	return time.Duration(d).String()
}

// Default returns the values used for keys missing from the configuration file
func Default() *Config {
	return &Config{
		Port:           80,
		Address:        "127.0.0.1",
		ReadTimeout:    Duration(12 * time.Second),
		WriteTimeout:   Duration(12 * time.Second),
		IdleTimeout:    Duration(60 * time.Second),
		MaxHeaderBytes: 1000000,

		DocumentRoot: "public",
		CacheRoot:    "cache",
		LogRoot:      "logs",
		UploadsRoot:  "private/uploads",

		Aliases:  map[string]string{},
		DenyFrom: []string{},

		StatusAllowFrom: []string{"127.0.0.0/8", "::1/128"},

		ServerSignature:        "Golang Preprocessor for Web Services v1.0",
		DirectoryListingScript: "/../private/scripts/DirectoryListing.gop",
		IPDeniedScript:         "/../private/scripts/ip_denied.gop",
		SessionCookieName:      "GopSession",
		GopHeadersSeparator:    "<!--GOP-HEADERS-S3P4R4T0R",
		LoggingLevel:           "basic",
		GopKeySalt:             PlaceholderSalt,

		RateLimitRate:     10,
		RateLimitPeriod:   20,
		RateLimitedScript: "/../private/scripts/ip_ratelimited.gop",

		BuildTimeout:     Duration(60 * time.Second),
		ExecutionTimeout: Duration(6 * time.Second),
	}
}

// Load reads the configuration file at path over the defaults and validates it
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading configuration: %w", err)
	}

	c := Default()
	if err := json.Unmarshal(raw, c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

func (c *Config) validate() error {
	var problems []error
	check := func(ok bool, format string, args ...any) {
		if !ok {
			problems = append(problems, fmt.Errorf(format, args...))
		}
	}

	check(c.Port > 0 && c.Port < 65536, "Port %d is not between 1 and 65535", c.Port)
	check(c.ReadTimeout >= 0 && c.WriteTimeout >= 0 && c.IdleTimeout >= 0, "timeouts cannot be negative")
	check(c.BuildTimeout > 0 && c.ExecutionTimeout > 0, "BuildTimeout and ExecutionTimeout must be positive")
	check(c.MaxHeaderBytes >= 0, "MaxHeaderBytes cannot be negative")

	info, err := os.Stat(c.DocumentRoot)
	check(err == nil && info.IsDir(), "DocumentRoot %q is not a directory", c.DocumentRoot)
	for name, dir := range map[string]string{"CacheRoot": c.CacheRoot, "LogRoot": c.LogRoot, "UploadsRoot": c.UploadsRoot} {
		check(dir != "", "%s cannot be empty", name)
	}

	networks := func(key string, cidrs []string) (parsed []*net.IPNet) {
		for _, cidr := range cidrs {
			_, network, err := net.ParseCIDR(cidr)
			check(err == nil, "%s entry %q is not a CIDR range like 10.1.2.0/24", key, cidr)
			if err == nil {
				parsed = append(parsed, network)
			}
		}
		return parsed
	}
	c.DeniedNetworks = networks("DenyFrom", c.DenyFrom)
	c.StatusNetworks = networks("StatusAllowFrom", c.StatusAllowFrom)

	switch c.LoggingLevel {
	case "debug", "basic", "error", "errors":
	default:
		check(false, "LoggingLevel %q is not debug, basic or error", c.LoggingLevel)
	}
	check(c.SessionCookieName != "" && !strings.ContainsAny(c.SessionCookieName, " \t\r\n;,="), "SessionCookieName %q is not a valid cookie name", c.SessionCookieName)
	check(strings.TrimSpace(c.GopHeadersSeparator) != "" && !strings.ContainsAny(c.GopHeadersSeparator, "\r\n"), "GopHeadersSeparator must be a non-empty single line")
	if c.RateLimitEnabled {
		check(c.RateLimitRate > 0 && c.RateLimitPeriod > 0, "RateLimitRate and RateLimitPeriod must be positive")
	}

	if c.GopKeySalt == PlaceholderSalt || c.GopKeySalt == "" {
		c.GopKeySalt = randomSalt()
		c.Warnings = append(c.Warnings, "GopKeySalt is not set; using a random salt for this run, so upload keys change on restart")
	}

	return errors.Join(problems...)
}

// Key derives a secret for one purpose (signing sessions, naming upload fields) from GopKeySalt
func (c *Config) Key(purpose string) []byte {
	mac := hmac.New(sha256.New, []byte(c.GopKeySalt))
	mac.Write([]byte(purpose))
	return mac.Sum(nil)
}

func randomSalt() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand does not fail on supported platforms
	}
	return hex.EncodeToString(b)
}

// Contains reports whether ip is in one of networks
func Contains(networks []*net.IPNet, ip net.IP) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
