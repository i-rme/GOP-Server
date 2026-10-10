package pages

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// compilerError matches the first located message of the go command: file:line:column: message
var compilerError = regexp.MustCompile(`(?m)^([^:\n]+):(\d+):\d+: (.+)$`)

// Explain turns the go command's output for a failed .gop build into a short explanation
// with the lines of the script around the first error. Output it cannot read is returned as is.
func Explain(output string) string {
	match := compilerError.FindStringSubmatch(output)
	if match == nil {
		return output
	}
	file, message := match[1], match[3]
	line, _ := strconv.Atoi(match[2])
	source, err := os.ReadFile(file)
	if err != nil {
		return output
	}

	var b strings.Builder
	fmt.Fprintf(&b, "The compiler detected an error on line %d.\n\n%s\n\nThe code related to the error is shown below:\n\t\n", line, message)
	for i, text := range strings.Split(strings.TrimSuffix(string(source), "\n"), "\n") {
		if n := i + 1; n > line-5 && n < line+5 {
			fmt.Fprintf(&b, "%d.   %s\n", n, strings.TrimSuffix(text, "\r"))
		}
	}
	return b.String()
}
