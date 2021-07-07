package parser

import (
	"fmt"
	"pfg/src/server/config"
	"pfg/src/server/logs"
	"regexp"
	"strings"
)

// HasGopTags checks if a file contains Go embeded using <!go tags
func HasGopTags(content string) bool {
	r, _ := regexp.Compile(`^<!go(\r\n|\s|\n)(.*)`) //GOP tag followed by new line or white space
	return r.MatchString(content)
}

// RemoveTags returns the source without the <!go tags
func RemoveTags(content string) string {
	r := regexp.MustCompile(`<!go(\r\n|\s|\n)`) //GOP tag followed by new line or white space
	return r.ReplaceAllString(content, "${1}")
}

// GetImports returns the imports from a gop file	import ()
func GetImports(content string) string {
	/*
		Imports regex explained
		This regex is composed of two regex between an OR ( | )
		The first part matches the Go import ""
			((\s)+import(\s)*"[^"]+"(;)?)+
		The second part matches the Go import ()
			(\s)+import(\s)*\(((\s)*"[^"]+"(;)?)+(\s)*\)(;)?
		Features supported:
			- multiple imports
			- trailing semicolon on lines and imports
			- spaces and new lines between symbols
			- special characters like on packages names
	*/
	r := regexp.MustCompile(`(((\s)+import(\s)*"[^"]+"(;)?)+|(\s)+import(\s)*\(((\s)*(_ )?"[^"]+"(;)?)+(\s)*\)(;)?)`)
	match := r.FindStringSubmatch(content)

	if len(match) == 0 { //No matches, no imports
		return ""
	}

	return match[0]
}

// RemoveImports returns the source without the import ()
func RemoveImports(content string) string {
	r := regexp.MustCompile(`(((\s)+import(\s)*"[^"]+"(;)?)+|(\s)+import(\s)*\(((\s)*(_ )?"[^"]+"(;)?)+(\s)*\))`)
	return r.ReplaceAllString(content, "")
}

// SplitOutputAndWorkDirectory accepts a go run --work output and returns the output and the Working directory in separated values
func SplitOutputAndWorkDirectory(combinedOutput string) (string, string) {

	r := regexp.MustCompile(`WORK=([^\n]*)\n`)
	match := r.FindStringSubmatch(combinedOutput) //Stores the working directory in match[1]

	if len(match) == 0 { //No matches, no WORK line, error from the compiler command
		logs.WriteError("Unexpected error in SplitOutputAndWorkDirectory")
		return combinedOutput, ""
	}

	output := strings.Split(combinedOutput, match[0])[1] // Removes the working directory line from the ouput

	return output, match[1]
}

// GetRequire returns the dependencies (one each time)
func GetRequire(content string) string {
	r := regexp.MustCompile(`(\s)require\s"([^"]+)"(;)?`) // Matches require dependencies
	match := r.FindStringSubmatch(content)

	if len(match) == 0 {
		return ""
	}

	return match[2]
}

// RemoveRequire returns the source without the first require replaced by the dependency source (one each time)
func RemoveRequire(content, replacement string) string {

	r := regexp.MustCompile(`(\s)require\s"([^"]+)"(;)?`)
	found := r.FindString(content)
	if found != "" {
		return strings.Replace(content, found, replacement, 1) //Replaces only first occurence
	}

	return content
}

// SplitOutputAndHeaders separates the headers and the execution output
func SplitOutputAndHeaders(combinedOutput string) (string, string) {

	o := strings.Split(combinedOutput, config.GopHeadersSeparator+" ")
	buildOutput := o[0]
	headersOutput := o[1]

	return buildOutput, headersOutput
}

// SplitFunctionsAndMain is used to separate functions definitions (if any) from the main function
func SplitFunctionsAndMain(sourceCode string) (string, string) {
	lastFunctionIndex := findLastFunctionIndex(sourceCode)

	functionEndIndex := 0 // At the start there are no functions

	if lastFunctionIndex != -1 { // There is some function definition

		functionEndIndex = findFunctionEndIndex(sourceCode[lastFunctionIndex:])

		if functionEndIndex != -1 { // The function ends eventually

			functionEndIndex += lastFunctionIndex // Relative value to absolute

		} else { // The function never ends
			fmt.Println("Malformed function definition")
			return "", ""
		}

	}
	functions := sourceCode[:functionEndIndex]
	main := sourceCode[functionEndIndex:]

	return functions, main
}

// findLastFunctionIndex checks if functions are defined on the source code, then returns the last occurrence of function definition
func findLastFunctionIndex(content string) int {
	r := regexp.MustCompile(`(\s|\n)func [^(]+\(`) // Matches function definition
	match := r.FindAllStringIndex(content, 128)
	matchNum := len(match)

	if matchNum == 0 {
		return -1
	}

	return match[matchNum-1][0]
}

//findFunctionEndIndex tries to find where a function definition ends, this is a task that regex is unable to perform (See nested parenthesis regex problem)
func findFunctionEndIndex(content string) int {
	nestedDepth := 0
	lastFunctionIndex := -1

	for index, rune := range content {
		character := string(rune)

		if character == "{" {
			nestedDepth++
		} else if character == "}" {
			nestedDepth--
			lastFunctionIndex = index + 1
			if nestedDepth == 0 {
				return lastFunctionIndex
			}
		}
	}
	return lastFunctionIndex
}
