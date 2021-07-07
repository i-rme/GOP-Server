package handler

import (
	"fmt"
	"net"
	"net/http"
	"pfg/src/controller"
	"pfg/src/handler/errors"
	"pfg/src/repository/filesystem"
	"pfg/src/server/config"
	"pfg/src/server/logs"
)

// Handle tries to handle the request
func Handle(w http.ResponseWriter, r *http.Request) {

	urlPath := GetURLPath(r) // Obtains the url from the request

	if IsDenied(r) { // Check if remote IP is on the denylist
		Denied(w, r)
		return
	}

	if config.RateLimitEnabled {

		ipAddress, _, _ := net.SplitHostPort(r.RemoteAddr) // Removes the port from the IP

		if IsRateLimited(ipAddress, urlPath) { // Check if remote IP is on the denylist
			RateLimited(w, r)
			return
		}

		addRequestRateLimit(ipAddress)

	}

	if IsAlias(urlPath) { // Handle alias using Alias handler
		Alias(w, r)
		return
	}

	logs.WriteRequest(r.RemoteAddr, r.Method, r.Host, r.RequestURI, r.Proto)
	w.Header().Set("Server", config.ServerSignature)

	params := ParseParameters(r) // GET, POST, COOKIE, SERVER

	var cached, output, _headers string
	var status int

	switch filesystem.Extension(urlPath) {
	case ".go":
		output, status, cached = controller.RunGo(urlPath, params)
	case ".gop":
		output, _headers, status, cached = controller.RunGop(urlPath, params)
		status = AddHeaders(w, _headers, status)
	case "":
		output, _headers, status, cached = controller.RunIndex(urlPath, params)
		status = AddHeaders(w, _headers, status)
	default:
		RunStatic(urlPath, w, r)
	}

	switch filesystem.Extension(urlPath) {
	case ".go", ".gop", "":
		w.Header().Set("X-Cache-Status", cached)
		w.WriteHeader(status)
		fmt.Fprint(w, output)
	}

}

// RunStatic tries to serve static files
func RunStatic(urlPath string, w http.ResponseWriter, r *http.Request) {
	sourcePath := config.DocumentRoot + "/" + urlPath

	if !filesystem.Exists(sourcePath) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, errors.Render(errors.NotFound, urlPath))
	} else {
		fs := http.FileServer(http.Dir("./" + config.DocumentRoot))
		fs.ServeHTTP(w, r)
	}
}
