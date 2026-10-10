package parser

import (
	"regexp"
	"strings"
)

var workLine = regexp.MustCompile(`WORK=([^\n]*)\n`)

// SplitOutputAndWorkDirectory accepts a go run --work output and returns the output and the working directory.
// The directory is empty when the output has no WORK line, which means the go command itself failed.
func SplitOutputAndWorkDirectory(combinedOutput string) (string, string) {
	match := workLine.FindStringSubmatchIndex(combinedOutput)
	if match == nil {
		return combinedOutput, ""
	}
	return combinedOutput[match[1]:], combinedOutput[match[2]:match[3]]
}

// SplitOutputAndHeaders separates the script output from the JSON headers printed after the separator.
// The last separator wins, so a script printing the separator cannot hide its real headers;
// a script that exits before printing any (os.Exit, panic) gets no headers.
func SplitOutputAndHeaders(combinedOutput, separator string) (string, string) {
	i := strings.LastIndex(combinedOutput, separator+" ")
	if i < 0 {
		return combinedOutput, "[]"
	}
	return combinedOutput[:i], combinedOutput[i+len(separator)+1:]
}
