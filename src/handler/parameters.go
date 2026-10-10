package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/i-rme/gop-server/src/repository/filesystem"
)

// parameters extracts GET, POST & Cookie parameters from a request, then returns them as JSON
func (h *Handler) parameters(r *http.Request) string {
	r.ParseForm() //ParseForm populates r.Form and r.PostForm. (GET and POST parameters)

	GET := parseGET(r) //TO-DO Remove POST parameters from GET, as both are included
	POST := parsePOST(r)
	COOKIE := parseCOOKIE(r)
	SERVER := h.parseSERVER(r)
	SESSION := h.parseSESSION(r)
	HEADER := parseHEADER(r)

	h.parseMULTIPARTS(r)

	parameters := []map[string]string{GET, POST, COOKIE, SERVER, SESSION, HEADER}
	parametersJSON, _ := json.Marshal(parameters) //Parse GET map as a JSON byte[]

	return string(parametersJSON)
}

// parseGET extracts GET parameters from a request, then returns them as map
func parseGET(r *http.Request) map[string]string {

	_map := make(map[string]string) // Creating empty map

	for key, value := range r.Form { //Converts map[string][]string to map[string]string
		_map[key] = value[0]
	}

	return _map
}

// parsePOST extracts POST parameters from a request, then returns them as map
func parsePOST(r *http.Request) map[string]string {

	_map := make(map[string]string) // Creating empty map

	for key, value := range r.PostForm { //Converts map[string][]string to map[string]string
		_map[key] = value[0]
	}

	return _map
}

// parseCOOKIE extracts Cookies from a request, then returns them as map
func parseCOOKIE(r *http.Request) map[string]string {

	_map := make(map[string]string) // Creating empty map

	for _, cookie := range r.Cookies() {
		_map[cookie.Name] = cookie.Value
	}

	return _map
}

// parseSERVER prepares SERVER parameters, then returns them as JSON
func (h *Handler) parseSERVER(r *http.Request) map[string]string {

	_map := make(map[string]string) // Creating empty map

	currentPath, _ := os.Getwd() //rooted path name corresponding to the current directory

	parsedURI, _ := url.Parse(r.RequestURI) // Parses URI and extracts path, rawquery and query values

	ip, port, _ := net.SplitHostPort(r.RemoteAddr) // Removes the port from the IP

	if runtime.GOOS == "windows" { //Reverse slashes if Windows is detected
		_map["DOCUMENT_ROOT"] = currentPath + "\\" + h.config.DocumentRoot
		_map["SCRIPT_PATH"] = currentPath + "\\" + h.config.DocumentRoot + "\\" + filesystem.Directory(urlPath(r))
		_map["CURRENT_PATH"] = currentPath
	} else {
		_map["DOCUMENT_ROOT"] = currentPath + "/" + h.config.DocumentRoot
		_map["SCRIPT_PATH"] = currentPath + "/" + h.config.DocumentRoot + "/" + filesystem.Directory(urlPath(r))
		_map["CURRENT_PATH"] = "/" + currentPath
	}

	_map["DOCUMENT_ROOT_RELATIVE"] = h.config.DocumentRoot
	_map["SCRIPT_PATH_RELATIVE"] = "/" + filesystem.Directory(urlPath(r))
	_map["GOP_SELF"] = urlPath(r)
	_map["HTTP_REFERER"] = r.Referer()
	_map["HTTP_USER_AGENT"] = r.UserAgent()
	_map["REMOTE_ADDR"] = ip
	_map["REMOTE_PORT"] = port
	_map["REQUEST_METHOD"] = r.Method
	_map["SCRIPT_NAME"] = urlPath(r)
	_map["SCRIPT_BASENAME"] = filepath.Base(urlPath(r))
	_map["SERVER_NAME"] = r.Host
	_map["SERVER_PROTOCOL"] = r.Proto
	_map["REQUEST_URI"] = parsedURI.String()
	_map["REQUEST_PATH"] = parsedURI.Path
	_map["RAW_QUERY"] = parsedURI.RawQuery
	_map["FILE_UPLOAD_KEY"] = h.fileUploadKey(r)
	_map["SIGNATURE"] = h.config.ServerSignature
	_map["SESSION_COOKIE_NAME"] = h.config.SessionCookieName
	_map["LOGGING_LEVEL"] = h.config.LoggingLevel

	return _map
}

// parseSESSION prepares SERVER parameters, then returns them as JSON
func (h *Handler) parseSESSION(r *http.Request) map[string]string {

	var SESSION = make(map[string]string) // Prepares the slice of maps variable containing all the parameters

	for _, cookie := range r.Cookies() {
		if cookie.Name == h.config.SessionCookieName {
			sessionEncoded := cookie.Value
			sessionJSON, _ := base64.StdEncoding.DecodeString(sessionEncoded)

			sessionBytes := []byte(sessionJSON)    // JSON string to byte array
			json.Unmarshal(sessionBytes, &SESSION) // JSON to slice of maps
		}
	}

	return SESSION
}

// parseHEADER extracts Headers from a request, then returns them as map
func parseHEADER(r *http.Request) map[string]string {

	_map := make(map[string]string) // Creating empty map

	for header, value := range r.Header {
		_map[header] = value[0]
	}

	return _map
}

// parseMULTIPARTS extracts MultipartForm parameters from a request, then returns them
func (h *Handler) parseMULTIPARTS(r *http.Request) {

	if r.Method == "POST" {
		r.ParseMultipartForm(1 << 20) // upload file size maximum: 1MB

		fileUploadKey := h.fileUploadKey(r)

		file, handler, err := r.FormFile(fileUploadKey)
		if err != nil {
			return // Maybe there is no file upload
		}
		defer file.Close()

		h.log.Info("file uploaded", "name", handler.Filename, "size", handler.Size, "header", handler.Header)

		tempFile, err := os.CreateTemp(h.config.UploadsRoot, fileUploadKey+"_*_"+handler.Filename)
		if err != nil {
			h.log.Error("saving upload", "error", err)
			return
		}
		defer tempFile.Close()

		if _, err := io.Copy(tempFile, file); err != nil {
			h.log.Error("saving upload", "error", err)
		}

	}

}

// addHeaders sets the headers a script printed and returns the status it chose, if any
func addHeaders(w http.ResponseWriter, _headers string, status int) int {
	headers := make([]string, 0)           // Prepares the slice
	headersBytes := []byte(_headers)       // JSON string to byte array
	json.Unmarshal(headersBytes, &headers) // JSON to slice of maps

	for _, header := range headers {
		h := strings.Split(header, ": ")
		if len(h) == 2 {
			w.Header().Set(h[0], h[1])
		} else {
			status, _ = strconv.Atoi(h[0])
		}

	}

	return status
}

// fileUploadKey is the form field name a request's script accepts an upload under
func (h *Handler) fileUploadKey(r *http.Request) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(r.Host+r.UserAgent()+urlPath(r)+h.config.GopKeySalt)))
}
