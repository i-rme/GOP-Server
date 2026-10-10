package parser

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

/*
	legacy_test.go keeps the regex-based translation this package used before
	the tokenizer, trimmed of logging. The examples test checks that both
	produce the same program for every script the old code accepted.
*/

func legacyProcess(filePath string, options Options) (string, bool) {
	imports := ""
	for _, spec := range runtimeImports(options) {
		imports += "import " + importLine(spec) + "\n"
	}
	_imports, _functions, _content, valid := legacyDependency(filePath)
	return "package main\r\n" + imports + _imports + "\n" + runtimeConstants(options) +
		"\r\nfunc main() {" + _content + "\r\n__housekeeping()\r\n\r\n}\r\n" +
		runtimeCode(options) + _functions, valid
}

// runtimeCode is the runtime without its imports
func runtimeCode(options Options) string {
	code := ""
	for _, runtime := range runtimeFor(options) {
		code += runtime.code
	}
	return code
}

func legacyDependency(filePath string) (string, string, string, bool) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", "", false
	}
	content := string(raw)

	if !regexp.MustCompile(`^<!go(\r\n|\s|\n)(.*)`).MatchString(content) {
		return "", "", "", false
	}

	importsRegexp := regexp.MustCompile(`(((\s)+import(\s)*"[^"]+"(;)?)+|(\s)+import(\s)*\(((\s)*(_ )?"[^"]+"(;)?)+(\s)*\)(;)?)`)
	imports := ""
	if match := importsRegexp.FindStringSubmatch(content); len(match) > 0 {
		imports = match[0]
	}
	content = regexp.MustCompile(`(((\s)+import(\s)*"[^"]+"(;)?)+|(\s)+import(\s)*\(((\s)*(_ )?"[^"]+"(;)?)+(\s)*\))`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<!go(\r\n|\s|\n)`).ReplaceAllString(content, "${1}")

	functions, content := legacySplitFunctionsAndMain(content)

	requireRegexp := regexp.MustCompile(`(\s)require\s"([^"]+)"(;)?`)
	for {
		match := requireRegexp.FindStringSubmatch(content)
		if len(match) == 0 {
			break
		}
		_imports, _functions, _content, valid := legacyDependency(filepath.Dir(filePath) + "/" + match[2])
		if !valid {
			return "", "", "", false
		}
		imports += _imports
		functions += _functions
		content = strings.Replace(content, match[0], _content, 1)
	}

	return imports, functions, content, true
}

func legacySplitFunctionsAndMain(sourceCode string) (string, string) {
	matches := regexp.MustCompile(`(\s|\n)func [^(]+\(`).FindAllStringIndex(sourceCode, 128)
	if len(matches) == 0 {
		return "", sourceCode
	}
	last := matches[len(matches)-1][0]

	depth, end := 0, -1
	for i, r := range sourceCode[last:] {
		if r == '{' {
			depth++
		} else if r == '}' {
			depth--
			end = i + 1
			if depth == 0 {
				break
			}
		}
	}
	if end == -1 {
		return "", ""
	}
	return sourceCode[:last+end], sourceCode[last+end:]
}
