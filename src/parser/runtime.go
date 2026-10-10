//go:build ignore

// The runtime of every .gop script. The parser embeds this file and adds it to the
// generated program, together with __headersSeparator.

package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Public Variables
var _GET, _POST, _COOKIE, _SERVER, _SESSION, _HEADER map[string]string

// Private Variables
var __sessionActive = false
var __status = 0              // HTTP status set by the script, 0 for the server's default
var __headers = [][2]string{} // response headers set by the script, in order

// Public Methods
func _htmlentities(input string) string {
	return html.EscapeString(input)
}
func _html_entity_decode(input string) string {
	return html.UnescapeString(input)
}
func _nl2br(input string) string {
	return strings.Replace(input, "\n", "<br>", -1)
}
func _echo(input ...interface{}) {
	fmt.Print(input...)
}

// _header adds a "Name: value" header, or sets the status when given a bare code like "404"
func _header(input string) {
	if name, value, ok := strings.Cut(input, ": "); ok {
		__headers = append(__headers, [2]string{name, value})
	} else if code, err := strconv.Atoi(strings.TrimSpace(input)); err == nil {
		__status = code
	}
}
func _redirect(input string) {
	_header("302")
	_header("Location: " + input)
}
func _status_code(input int) {
	__status = input
}
func _setcookie(name, value string) {
	_header("Set-Cookie: " + name + "=" + value)
}
func _session_start() {
	__sessionActive = true
}
func _setsession(name, value string) {
	if __sessionActive {
		_SESSION[name] = value
	}
}
func _session_destroy() {
	_SESSION = make(map[string]string)
	__sessionActive = true // an empty session tells the server to expire the cookie
}
func _base64_encode(plain string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(plain))
	return encoded
}
func _base64_decode(encoded string) string {
	plain, _ := base64.StdEncoding.DecodeString(encoded)
	return string(plain)
}
func _require_basic_auth(message string) {
	authHeader := "WWW-Authenticate: Basic realm=\"" + message + "\""
	_status_code(401) // 401 Unauthorized
	_header(authHeader)
}
func _parseAuthentication(authorization string) (string, string) {
	user, pass := splitByColon(_base64_decode(extractBasicAuthHeader(authorization)))
	return user, pass
}
func _json_encode(plain interface{}) string {
	encoded, _ := json.Marshal(plain)
	return string(encoded)
}
func _json_decode(encoded string) interface{} {
	//TO-DO
	return encoded
}
func _string_to_int(input string) int {
	integer, err := strconv.Atoi(input)

	if err != nil {
		return 0
	}

	return integer
}
func _int_to_string(input int) string {
	string := strconv.Itoa(input)
	return string
}
func _isset(input string) bool {
	if input == "" {
		return false
	}
	return true
}
func _date(format string) string {
	dt := time.Now()
	return dt.Format(format)
}
func _time(format string) string {
	return _date(format)
}
func substr(string string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(string) {
		end = len(string)
	}
	return string[start:end]
}

// Private Methods

// __housekeeping ends the output with the trailer the server reads the response from:
// the separator, a space and {"status":N,"headers":[[name,value],...],"session":{...}} on one line.
// The session is null unless the script started or destroyed one; the server signs it into the cookie.
func __housekeeping() {
	var session map[string]string
	if __sessionActive {
		session = _SESSION
		if session == nil {
			session = map[string]string{}
		}
	}
	trailer, _ := json.Marshal(struct {
		Status  int               `json:"status"`
		Headers [][2]string       `json:"headers"`
		Session map[string]string `json:"session"`
	}{__status, __headers, session})
	fmt.Fprintln(os.Stdout, __headersSeparator, string(trailer))
}

// TO-DO Move it
func extractBasicAuthHeader(input string) string {
	r := regexp.MustCompile("Basic (.*)") // Matches Basic Auth Base64
	match := r.FindStringSubmatch(input)

	if len(match) == 0 {
		return ""
	}

	return match[1]
}
func splitByColon(input string) (string, string) {
	r := regexp.MustCompile("(.*):(.*)") // Matches each part
	match := r.FindStringSubmatch(input)

	if len(match) == 0 {
		return "", ""
	}

	return match[1], match[2]
}

func init() {

	parametersJSON := flag.String("PARAMETERS", "[]", "parameters") // Receives the PARAMETERS values as JSON in a launch flag
	flag.Parse()

	var parameters []map[string]string           // Prepares the slice of maps variable containing all the parameters
	parametersBytes := []byte(*parametersJSON)   // JSON string to byte array
	json.Unmarshal(parametersBytes, &parameters) // JSON to slice of maps

	if len(parameters) > 0 { // Avoid crashing if parameters are not passed
		_GET = parameters[0]
		_POST = parameters[1]
		_COOKIE = parameters[2]
		_SERVER = parameters[3]
		_SESSION = parameters[4]
		_HEADER = parameters[5]
	}
}
