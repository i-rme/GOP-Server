package worker

import (
	"os/exec"
	"pfg/src/parser"
	"pfg/src/server/config"
	"runtime"
	"strings"
)

// BuildAndRun calls the Golang compiler with less privileges
func BuildAndRun(inputPath, parameters string) (string, string, bool) {

	if runtime.GOOS == "windows" || !config.RunScriptsAsNobody {
		return buildAndRunGeneric(inputPath, parameters)
	} else {
		//return buildAndRunLinux(inputPath, parameters)
		return buildAndRunGeneric(inputPath, parameters)
	}

}

// buildAndRunGeneric calls the Golang compiler from windows
func buildAndRunGeneric(inputPath, parameters string) (string, string, bool) {

	command := exec.Command("go", "run", "--work", inputPath, `-PARAMETERS=`+parameters)
	outputBytes, err := command.CombinedOutput()
	output := strings.TrimSuffix(string(outputBytes), "\n")
	output, workingDirectory := parser.SplitOutputAndWorkDirectory(output)

	if err != nil {
		return err.Error() + "\n" + output, workingDirectory, false
	}

	return output, workingDirectory, true
}

// buildAndRunLinux calls the Golang compiler as unprivileged linux user
func buildAndRunLinux(inputPath, parameters string) (string, string, bool) {

	//Preparing parameters
	// Sorry fot this escaping mess, but it worked
	parameters = strings.ReplaceAll(parameters, `\"`, ``)
	parameters = strings.ReplaceAll(parameters, ` `, `\ `)
	parameters = strings.ReplaceAll(parameters, `"`, `\"`)
	parameters = strings.ReplaceAll(parameters, `/`, `\/`)
	parameters = strings.ReplaceAll(parameters, `;`, `\;`)
	parameters = strings.ReplaceAll(parameters, `(`, `\(`)
	parameters = strings.ReplaceAll(parameters, `)`, `\)`)
	parameters = strings.ReplaceAll(parameters, `,`, `\,`)

	// Runs as nobody
	command := exec.Command(`setpriv`, `--no-new-privs`, `--reuid=nobody`, `/bin/bash`, `-c`, `GOCACHE=/tmp/GOP/.cache/go-build GOENV=/tmp/GOP/.config/go/env GOPATH=/tmp/GOP/go go run --work `+inputPath+` -PARAMETERS=`+parameters)

	outputBytes, err := command.CombinedOutput()
	output := strings.TrimSuffix(string(outputBytes), "\n")
	output, workingDirectory := parser.SplitOutputAndWorkDirectory(output)

	if err != nil {
		return err.Error() + "\n" + output, workingDirectory, false
	}

	return output, workingDirectory, true
}

// build calls the Golang compiler (DEPRECATED)
func build(inputPath string, outputPath string) (string, bool) {

	command := exec.Command("go", "build", "-o", inputPath, outputPath)
	outputBytes, err := command.CombinedOutput()
	output := strings.TrimSuffix(string(outputBytes), "\n")

	if err != nil { // TO-DO handle the error
		return err.Error() + "\n" + output, false
	}

	return output, true
}
