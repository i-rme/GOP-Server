package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

// maxUpload is the largest file a script accepts through FILE_UPLOAD_KEY
const maxUpload = 1 << 20

// maxMemory is how much of a multipart body is kept in memory; the rest goes to temporary files
const maxMemory = 1 << 20

// parameters returns what the script gets in -PARAMETERS: the JSON list
// [_GET, _POST, _COOKIE, _SERVER, _SESSION, _HEADER]. urlPath is the script's URL path.
// A file sent under FILE_UPLOAD_KEY is saved to UploadsRoot on the way.
func (h *Handler) parameters(r *http.Request, urlPath string) (string, error) {
	if err := parseBody(r); err != nil {
		return "", err
	}
	h.saveUpload(r, urlPath)

	cookies := make(map[string]string)
	for _, cookie := range r.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}

	parameters, err := json.Marshal([]map[string]string{
		first(r.URL.Query()),
		first(r.PostForm),
		cookies,
		h.server(r, urlPath),
		h.sessions.read(r),
		first(r.Header),
	})
	return string(parameters), err
}

// parseBody parses a urlencoded or multipart body into r.PostForm
func parseBody(r *http.Request) error {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaType == "multipart/form-data" {
		return r.ParseMultipartForm(maxMemory)
	}
	return r.ParseForm()
}

// first keeps the first value of each key
func first(values map[string][]string) map[string]string {
	m := make(map[string]string, len(values))
	for key, list := range values {
		if len(list) > 0 {
			m[key] = list[0]
		}
	}
	return m
}

// server builds the _SERVER map for a script answering urlPath
func (h *Handler) server(r *http.Request, urlPath string) map[string]string {
	ip, port, _ := net.SplitHostPort(r.RemoteAddr)
	requestURI, _ := url.Parse(r.RequestURI)

	// A path without an extension is taken for a directory, as the script's location
	scriptDir := urlPath
	if path.Ext(urlPath) != "" {
		scriptDir = path.Dir(urlPath)
	}
	join := func(elem ...string) string { // keeps trailing slashes, as scripts have always seen them
		if runtime.GOOS == "windows" {
			return filepath.FromSlash(joinRaw(elem))
		}
		return joinRaw(elem)
	}

	documentRoot := h.config.DocumentRoot
	if !filepath.IsAbs(documentRoot) {
		documentRoot = join(h.workDir, documentRoot)
	}

	return map[string]string{
		"DOCUMENT_ROOT":          documentRoot,
		"SCRIPT_PATH":            join(documentRoot, scriptDir),
		"CURRENT_PATH":           h.workDir,
		"DOCUMENT_ROOT_RELATIVE": h.config.DocumentRoot,
		"SCRIPT_PATH_RELATIVE":   "/" + scriptDir,
		"GOP_SELF":               urlPath,
		"SCRIPT_NAME":            urlPath,
		"SCRIPT_BASENAME":        path.Base(urlPath),
		"HTTP_REFERER":           r.Referer(),
		"HTTP_USER_AGENT":        r.UserAgent(),
		"REMOTE_ADDR":            ip,
		"REMOTE_PORT":            port,
		"REQUEST_METHOD":         r.Method,
		"SERVER_NAME":            r.Host,
		"SERVER_PROTOCOL":        r.Proto,
		"REQUEST_URI":            requestURI.String(),
		"REQUEST_PATH":           requestURI.Path,
		"RAW_QUERY":              requestURI.RawQuery,
		"FILE_UPLOAD_KEY":        h.fileUploadKey(r, urlPath),
		"SIGNATURE":              h.config.ServerSignature,
		"SESSION_COOKIE_NAME":    h.config.SessionCookieName,
		"LOGGING_LEVEL":          h.config.LoggingLevel,
	}
}

// joinRaw joins path elements with slashes without cleaning them
func joinRaw(elem []string) string {
	joined := elem[0]
	for _, e := range elem[1:] {
		joined += "/" + e
	}
	return joined
}

// fileUploadKey is the form field a script's page accepts an upload under. It is a MAC of
// the client and the script, so only a page the server rendered for this client knows it.
func (h *Handler) fileUploadKey(r *http.Request, urlPath string) string {
	mac := hmac.New(sha256.New, h.uploadKey)
	io.WriteString(mac, r.Host+"\n"+r.UserAgent()+"\n"+urlPath)
	return hex.EncodeToString(mac.Sum(nil))
}

// saveUpload stores the file sent under FILE_UPLOAD_KEY, if any, under a random name
func (h *Handler) saveUpload(r *http.Request, urlPath string) {
	if r.MultipartForm == nil {
		return
	}
	file, header, err := r.FormFile(h.fileUploadKey(r, urlPath))
	if errors.Is(err, http.ErrMissingFile) {
		return
	}
	if err != nil {
		h.log.Warn("reading upload", "error", err)
		return
	}
	defer file.Close()
	if header.Size > maxUpload {
		h.log.Warn("upload too large", "name", header.Filename, "size", header.Size)
		return
	}

	stored, err := os.CreateTemp(h.config.UploadsRoot, "upload-*")
	if err == nil {
		_, err = io.Copy(stored, file)
		if closeErr := stored.Close(); err == nil {
			err = closeErr
		}
	}
	if err != nil {
		h.log.Error("saving upload", "error", err)
		return
	}
	h.log.Info("file uploaded", "name", header.Filename, "size", header.Size, "stored", stored.Name())
}
