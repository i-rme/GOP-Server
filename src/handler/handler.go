// Package handler answers HTTP requests: scripts, directories, static files and the built-in pages.
package handler

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/i-rme/gop-server/src/controller"
	"github.com/i-rme/gop-server/src/handler/pages"
	"github.com/i-rme/gop-server/src/repository/filesystem"
	"github.com/i-rme/gop-server/src/server/config"
	"github.com/i-rme/gop-server/src/server/logs"
)

// Handler serves the document root
type Handler struct {
	config     *config.Config
	log        *slog.Logger
	stats      *logs.Stats
	controller *controller.Controller
	limiter    *rateLimiter
	sessions   sessions
	uploadKey  []byte

	// Scripts named in the configuration, resolved once; they may live outside the document root
	listingScript, deniedScript, rateLimitedScript script
	aliases                                        []alias // longest prefix first
}

// script is a script to run and the URL path it is known by
type script struct {
	path    string // on disk
	urlPath string // without the leading slash, as scripts see it in _SERVER
}

type alias struct {
	prefix string
	target script
}

// New returns a Handler for the given configuration; stats counts the requests it serves
func New(c *config.Config, log *slog.Logger, stats *logs.Stats) (*Handler, error) {
	ctrl, err := controller.New(c, log)
	if err != nil {
		return nil, err
	}
	h := &Handler{
		config:     c,
		log:        log,
		stats:      stats,
		controller: ctrl,
		limiter:    newRateLimiter(c.RateLimitRate, time.Duration(c.RateLimitPeriod)*time.Second),
		sessions:   sessions{name: c.SessionCookieName, key: c.Key("session")},
		uploadKey:  c.Key("upload"),
	}

	configured := []struct {
		key, value string
		target     *script
		needed     bool
	}{
		{"DirectoryListingScript", c.DirectoryListingScript, &h.listingScript, true},
		{"IPDeniedScript", c.IPDeniedScript, &h.deniedScript, len(c.DenyFrom) > 0},
		{"RateLimitedScript", c.RateLimitedScript, &h.rateLimitedScript, c.RateLimitEnabled},
	}
	for _, s := range configured {
		*s.target = h.configuredScript(s.value)
		if s.needed && !filesystem.Exists(s.target.path) {
			return nil, fmt.Errorf("%s %q: no such file under DocumentRoot", s.key, s.value)
		}
	}
	for prefix, target := range c.Aliases {
		h.aliases = append(h.aliases, alias{prefix, h.configuredScript(target)})
	}
	sort.Slice(h.aliases, func(i, j int) bool { return len(h.aliases[i].prefix) > len(h.aliases[j].prefix) })
	return h, nil
}

// configuredScript resolves a script path from the configuration, relative to DocumentRoot.
// Configuration is trusted, so it may point outside the document root, as the shipped scripts do.
func (h *Handler) configuredScript(value string) script {
	return script{path: filepath.Join(h.config.DocumentRoot, value), urlPath: strings.TrimPrefix(value, "/")}
}

// ServeHTTP handles a request for anything under the document root
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logRequest(r)
	w.Header().Set("Server", h.config.ServerSignature)
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)

	if config.Contains(h.config.DeniedNetworks, net.ParseIP(ip)) {
		h.log.Warn("IP address is denied", "ip", ip)
		h.run(w, r, h.deniedScript)
		return
	}
	if h.config.RateLimitEnabled && !h.limiter.allow(ip, time.Now()) {
		h.log.Warn("IP address is rate limited", "ip", ip)
		h.run(w, r, h.rateLimitedScript)
		return
	}
	for _, a := range h.aliases {
		if strings.HasPrefix(r.URL.Path, a.prefix) {
			h.run(w, r, a.target)
			return
		}
	}

	urlPath := strings.TrimPrefix(r.URL.Path, "/")
	path, err := filesystem.ResolveInRoot(h.config.DocumentRoot, urlPath)
	if err != nil {
		h.notFound(w, urlPath)
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		h.notFound(w, urlPath)
		return
	}

	switch {
	case info.IsDir():
		if index := filepath.Join(path, "index.gop"); filesystem.Exists(index) {
			h.run(w, r, script{path: index, urlPath: urlPath})
		} else {
			h.run(w, r, script{path: h.listingScript.path, urlPath: urlPath})
		}
	case filepath.Ext(path) == ".gop" || filepath.Ext(path) == ".go":
		h.run(w, r, script{path: path, urlPath: urlPath})
	default:
		http.ServeFile(w, r, path)
	}
}

// run runs a script and writes its response. urlPath is the path the script answers for;
// a directory listing, for example, runs one script for many directories.
func (h *Handler) run(w http.ResponseWriter, r *http.Request, s script) {
	params, err := h.parameters(r, s.urlPath)
	if err != nil {
		h.log.Warn("bad request", "error", err)
		h.error(w, http.StatusBadRequest, pages.BadRequest, err.Error())
		return
	}
	response := h.controller.Run(s.path, s.urlPath, params)

	header := w.Header()
	for _, field := range response.Header {
		name, value := field[0], field[1]
		if !validHeader(name, value) {
			h.log.Warn("script set an invalid header", "script", s.urlPath, "name", name)
			continue
		}
		if http.CanonicalHeaderKey(name) == "Set-Cookie" {
			header.Add(name, value) // a script may set several cookies
		} else {
			header.Set(name, value)
		}
	}
	if response.Session != nil {
		http.SetCookie(w, h.sessions.cookie(response.Session, r.TLS != nil))
	}
	header.Set("X-Cache-Status", response.Cache)

	status := response.Status
	if status < 100 || status > 999 {
		h.log.Warn("script set an invalid status", "script", s.urlPath, "status", status)
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
	fmt.Fprint(w, response.Body)
}

// notFound answers a request for a path that is not in the document root
func (h *Handler) notFound(w http.ResponseWriter, urlPath string) {
	h.error(w, http.StatusNotFound, pages.NotFound, urlPath)
}

// error answers with an error page
func (h *Handler) error(w http.ResponseWriter, status int, kind pages.Kind, details string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprint(w, pages.Error(h.config.ServerSignature, kind, details))
}

// logRequest logs a request and counts it for the status page
func (h *Handler) logRequest(r *http.Request) {
	h.log.Info("request", "remote", r.RemoteAddr, "method", r.Method, "url", r.Host+r.RequestURI, "proto", r.Proto)
	h.stats.Add(time.Now())
}

// validHeader reports whether a script's header can be sent as is:
// a token for a name and no line breaks in the value
func validHeader(name, value string) bool {
	if name == "" || strings.ContainsAny(value, "\r\n\x00") {
		return false
	}
	for _, c := range name {
		if c > '~' || c <= ' ' || strings.ContainsRune(`"(),/:;<=>?@[\]{}`, c) {
			return false
		}
	}
	return true
}
