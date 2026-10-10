// Package handler answers HTTP requests: scripts, directories, static files and the built-in pages.
package handler

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/i-rme/gop-server/src/controller"
	"github.com/i-rme/gop-server/src/handler/errors"
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
	gopInfo    string
}

// New returns a Handler for the given configuration; stats counts the requests it serves
func New(c *config.Config, log *slog.Logger, stats *logs.Stats) (*Handler, error) {
	ctrl, err := controller.New(c, log)
	if err != nil {
		return nil, err
	}
	return &Handler{
		config:     c,
		log:        log,
		stats:      stats,
		controller: ctrl,
		limiter:    newRateLimiter(c.RateLimitRate, time.Duration(c.RateLimitPeriod)*time.Second),
		gopInfo:    gopInfoPage(c),
	}, nil
}

// ServeHTTP handles a request for anything under the document root
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := urlPath(r)
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)

	if r.URL.Path != h.config.IPDeniedScript && h.denied(ip) {
		h.log.Error("IP address is blocked for accessing this site", "ip", ip)
		r.URL.Path = h.config.IPDeniedScript
		h.ServeHTTP(w, r)
		return
	}

	if h.config.RateLimitEnabled && "/"+urlPath != h.config.RateLimitedScript && !h.limiter.allow(ip, time.Now()) {
		h.log.Error("IP address is rate limited", "ip", ip)
		r.URL.Path = h.config.RateLimitedScript
		h.ServeHTTP(w, r)
		return
	}

	if target, ok := h.alias(urlPath); ok {
		r.URL.Path = target
		h.ServeHTTP(w, r)
		return
	}

	h.logRequest(r)
	w.Header().Set("Server", h.config.ServerSignature)

	params := h.parameters(r)

	var response controller.Response
	switch filepath.Ext(urlPath) {
	case ".go":
		response = h.controller.RunGo(urlPath, params)
	case ".gop":
		response = h.controller.RunGop(urlPath, params)
	case "":
		response = h.controller.RunIndex(urlPath, params)
	default:
		h.serveStatic(w, r, urlPath)
		return
	}

	for _, header := range response.Header {
		w.Header().Set(header[0], header[1])
	}
	w.Header().Set("X-Cache-Status", response.Cache)
	w.WriteHeader(response.Status)
	fmt.Fprint(w, response.Body)
}

// denied reports whether ip is in one of the DenyFrom ranges
func (h *Handler) denied(ip string) bool {
	parsed := net.ParseIP(ip)
	for _, network := range h.config.DeniedNetworks {
		if network.Contains(parsed) {
			return true
		}
	}
	return false
}

// alias returns the script an aliased URL path maps to
func (h *Handler) alias(urlPath string) (string, bool) {
	for prefix, target := range h.config.Aliases {
		if strings.HasPrefix("/"+urlPath, prefix) {
			return target, true
		}
	}
	return "", false
}

// serveStatic serves a file from the document root
func (h *Handler) serveStatic(w http.ResponseWriter, r *http.Request, urlPath string) {
	if !filesystem.Exists(filepath.Join(h.config.DocumentRoot, urlPath)) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, errors.Render(h.config.ServerSignature, errors.NotFound, urlPath))
		return
	}
	http.FileServer(http.Dir(h.config.DocumentRoot)).ServeHTTP(w, r)
}

// logRequest logs a request and counts it for the status page
func (h *Handler) logRequest(r *http.Request) {
	h.log.Info("request", "remote", r.RemoteAddr, "method", r.Method, "url", r.Host+r.RequestURI, "proto", r.Proto)
	h.stats.Add(time.Now())
}

// urlPath is the request path without its leading slash
func urlPath(r *http.Request) string {
	return strings.TrimPrefix(r.URL.Path, "/")
}
