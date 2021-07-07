package errors

import (
	"bufio"
	"pfg/src/repository/filesystem"
	"pfg/src/server/logs"
	"regexp"
	"strconv"
	"strings"
)

//GetError gets a syntax error output and tries to obtain the source code that trigged it
func GetError(details string) string {

	_, fileName, line, _, explanation := ParseDetails(details)
	source, err := filesystem.Read(fileName)

	if err != nil {
		logs.WriteError("ERROR: ErrorExplainer was unable to parse the details of the error. Unhandled error?")
		source = "ERROR: ErrorExplainer was unable to parse the details of the error. Unhandled error?"
		return details
	}

	error := `The compiler detected an error on line ` + strconv.Itoa(line) + `.

` + explanation + `

The code related to the error is shown below:
	
`

	error += GetErrorLines(source, line) // Add source lines related

	return error
}

//GetErrorLines gets a syntax error output and tries to obtain the source code that trigged it
func GetErrorLines(source string, line int) string {

	errorLines := ""

	//Iterates over the source file
	currentLine := 1
	scanner := bufio.NewScanner(strings.NewReader(source))
	for scanner.Scan() {
		if currentLine > (line-5) && currentLine < (line+5) {
			errorLines += strconv.Itoa(currentLine) + ".   " + scanner.Text() + "\n"
		}
		currentLine++
	}

	return errorLines
}

// ParseDetails gets the console output and tries to return the source code that triggers the error
func ParseDetails(content string) (string, string, int, int, string) {
	/*
		Regex Fragment									| Meaning
		=========================================================
		([^:]+)											| Matches the first line of the output, just text
			   \n										| Matches a new line character
				 ([^:]+)								| Matches the name of the file
						:								| Matches : separator
						 ([0-9]+)						| Matches the line of the error
								 :						| Matches : separator
								  ([0-9]+)				| Matches the collumn of the error
										  : 			| Matches : separator
										    ([^\n]+)	| Matches the error text

	*/
	r := regexp.MustCompile(`([^:]+)\n([^:]+):([0-9]+):([0-9]+): ([^\n]+)`) // Matches require dependencies
	match := r.FindStringSubmatch(content)

	header := ""
	fileName := ""
	line := 0
	collumn := 0
	explanation := ""

	if len(match) == 0 {
		return header, fileName, line, collumn, explanation
	}

	header = match[1]
	fileName = match[2]
	line, _ = strconv.Atoi(match[3])    // String to Int
	collumn, _ = strconv.Atoi(match[4]) // String to Int
	explanation = match[5]

	return header, fileName, line, collumn, explanation
}
