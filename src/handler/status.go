package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/i-rme/gop-server/src/handler/pages"
	"github.com/i-rme/gop-server/src/server/config"
)

// ServerStatus serves /server-status/, and with ?data=true the request counts its chart polls
func (h *Handler) ServerStatus(w http.ResponseWriter, r *http.Request) {
	if !h.statusAllowed(w, r) {
		return
	}
	now := time.Now()
	if r.URL.Query().Get("data") == "true" {
		starts, counts := h.stats.Recent(now)
		type bucket struct {
			Start int64 `json:"start"`
			Count int   `json:"count"`
		}
		buckets := make([]bucket, len(starts))
		for i := range starts {
			buckets[i] = bucket{starts[i].Unix(), counts[i]}
		}
		slices.Reverse(buckets) // oldest first, as the chart draws them
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(buckets)
		return
	}
	h.render(w, func() error {
		return pages.Status(w, h.config.ServerSignature, net.JoinHostPort(h.config.Address, strconv.Itoa(h.config.Port)), now)
	})
}

// GopInfo serves /gop-info/, the configuration without anything secret
func (h *Handler) GopInfo(w http.ResponseWriter, r *http.Request) {
	if !h.statusAllowed(w, r) {
		return
	}
	c := h.config
	settings := []pages.Setting{
		{Name: "Port", Value: strconv.Itoa(c.Port)},
		{Name: "Address", Value: c.Address},
		{Name: "ReadTimeout", Value: c.ReadTimeout.String()},
		{Name: "WriteTimeout", Value: c.WriteTimeout.String()},
		{Name: "IdleTimeout", Value: c.IdleTimeout.String()},
		{Name: "MaxHeaderBytes", Value: strconv.Itoa(c.MaxHeaderBytes)},
		{Name: "BuildTimeout", Value: c.BuildTimeout.String()},
		{Name: "ExecutionTimeout", Value: c.ExecutionTimeout.String()},
		{Name: "DocumentRoot", Value: c.DocumentRoot},
		{Name: "CacheRoot", Value: c.CacheRoot},
		{Name: "LogRoot", Value: c.LogRoot},
		{Name: "ServerSignature", Value: c.ServerSignature},
		{Name: "DirectoryListingScript", Value: c.DirectoryListingScript},
		{Name: "IPDeniedScript", Value: c.IPDeniedScript},
		{Name: "SessionCookieName", Value: c.SessionCookieName},
		{Name: "GopHeadersSeparator", Value: c.GopHeadersSeparator},
		{Name: "RateLimitEnabled", Value: strconv.FormatBool(c.RateLimitEnabled)},
		{Name: "MySQLSupportEnabled", Value: strconv.FormatBool(c.MySQLSupportEnabled)},
		{Name: "RunScriptsAsNobody", Value: strconv.FormatBool(c.RunScriptsAsNobody)},
	}
	h.render(w, func() error { return pages.Info(w, c.ServerSignature, settings) })
}

// statusAllowed answers with 403 unless the client is in StatusAllowFrom
func (h *Handler) statusAllowed(w http.ResponseWriter, r *http.Request) bool {
	h.logRequest(r)
	w.Header().Set("Server", h.config.ServerSignature)
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if config.Contains(h.config.StatusNetworks, net.ParseIP(ip)) {
		return true
	}
	h.error(w, http.StatusForbidden, pages.Forbidden, r.URL.Path)
	return false
}

// render writes an HTML page
func (h *Handler) render(w http.ResponseWriter, page func() error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page(); err != nil {
		h.log.Error("rendering page", "error", err)
	}
}
