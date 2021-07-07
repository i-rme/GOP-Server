package parser

import (
	"pfg/src/repository/filesystem"
	"pfg/src/server/config"
	"pfg/src/server/logs"
)

// Process reads a GOP file and returns valid GO source resolving dependencies if needed
func Process(filePath string) (string, bool) {

	source := PackageDefinition
	imports := ImportsDefinition

	variables := VariablesDefinition
	functions := FunctionsDefinition
	mainContent := MainFunction

	if config.MySQLSupportEnabled {
		imports += ImportsDefinitionMySQL
		functions += FunctionsDefinitionMySQL

	}

	_imports, _functions, _content, valid := processDependency(filePath) //Work on dependencies
	imports += _imports
	functions += _functions
	mainContent += _content

	mainContent += MainFunctionEnd //Close main() function

	source += imports
	source += variables
	source += mainContent
	source += functions

	return source, valid
}

// processDependency reads a GOP file and returns imports, functions, content resolving dependencies if needed
func processDependency(filePath string) (string, string, string, bool) {

	if !filesystem.Exists(filePath) {
		return "", "", "", false // Dependency does not exist on filesystem
	}

	path := filesystem.Directory(filePath)
	content, err := filesystem.Read(filePath)

	if err != nil {
		logs.WriteError("ERROR: processDependency was unable to read the filePath")
		panic(err)
	}

	if !HasGopTags(content) {
		return "", "", "", false
	}

	imports := GetImports(content)
	content = RemoveImports(content)

	content = RemoveTags(content)

	functions, content := SplitFunctionsAndMain(content)

	for { // Do while new dependencies are found (until break)

		dependency := GetRequire(content)
		if dependency == "" {
			break
		}

		_imports, _functions, _content, _valid := processDependency(path + "/" + dependency) //Work on dependencies recursively
		if !_valid {
			return "", "", "", false
		}

		imports += _imports
		functions += _functions
		content = RemoveRequire(content, _content)
	}

	return imports, functions, content, true

}
