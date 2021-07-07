package parser

import "pfg/src/server/config"

var publicFunctions = `
//Public Methods
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
func _header(input string) {
	__responseHeader = append(__responseHeader, input)
}
func _redirect(input string) {
	_header("302")
	_header("Location: " + input)
}
func _status_code(input int) {
	inputString := strconv.Itoa(input)
	_header(inputString)
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
	if(input == ""){
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
func substr(string string, start, end int) string{
	if(start < 0){
		start = 0
	}
	if(end > len(string)){
		end = len(string)
	}
	return string[start:end]
}
`

var privateFunctions = `
//Private Methods
func __session_call() {
	if __sessionActive {
		sessionJSON, _ := json.Marshal(_SESSION) //Parse SESSION map as a JSON byte[]
		_setcookie("` + config.SessionCookieName + `", base64.StdEncoding.EncodeToString(sessionJSON))
	}
}
func __housekeeping() {

	// Required to handle sessions
	__session_call()

	// Required to add content type if missing
	var hasContentType = false

	for i := range __responseHeader {
		if strings.HasPrefix(__responseHeader[i], "Content-Type"){
			hasContentType = true
			break
		}
	}

	if !hasContentType {
		_header("Content-Type: text/html")
	}

	// Required to pass headers to the controller
	_headersJSON, _ := json.Marshal(__responseHeader)
	fmt.Println("` + config.GopHeadersSeparator + `", string(_headersJSON))
}

//TO-DO Move it
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
`

var mysqlFunctions = `
//Public MySQL Methods
func _mysqli_connect(host, user, password, database string) *sql.DB {
	connection, err := sql.Open("mysql", user+":"+password+"@tcp("+host+":3306)/"+database+"?timeout=2s")

	if err != nil {
		fmt.Println("DB CONNECTION ERROR")
		fmt.Println(err.Error())

		__housekeeping()
		os.Exit(0)
	}

	return connection
}

func _mysqli_query(connection *sql.DB, query string) *sql.Rows {
	result, err := connection.Query(query)

	//if ( err != nil && strings.HasPrefix(err.Error(), "dial") ) {	// Only crash if database is down, not if there is a query error

	if err != nil {
		fmt.Println("DB QUERY ERROR")
		fmt.Println(err)

		__housekeeping()
		os.Exit(0)
	}

	return result

}
`
