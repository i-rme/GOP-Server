package parser

var middlewareFunction = `
func init() {

	parametersJSON := flag.String("PARAMETERS", "[]", "parameters") // Receives the PARAMETERS values as JSON in a launch flag
	flag.Parse()

	var parameters []map[string]string           // Prepares the slice of maps variable containing all the parameters
	parametersBytes := []byte(*parametersJSON)   // JSON string to byte array
	json.Unmarshal(parametersBytes, &parameters) // JSON to slice of maps

	if len(parameters) > 0 {	// Avoid crashing if parameters are not passed
		_GET = parameters[0]
		_POST = parameters[1]
		_COOKIE = parameters[2]
		_SERVER = parameters[3]
		_SESSION = parameters[4]
		_HEADER = parameters[5]
	}

	_ = _GET     // Dummy assignment to avoid variable not used
	_ = _POST    // Dummy assignment to avoid variable not used
	_ = _COOKIE  // Dummy assignment to avoid variable not used
	_ = _SERVER  // Dummy assignment to avoid variable not used
	_ = _SESSION // Dummy assignment to avoid variable not used
	_ = _HEADER  // Dummy assignment to avoid variable not used

}`
