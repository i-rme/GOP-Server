package controller

import (
	"pfg/src/repository/filesystem"
	"pfg/src/server/config"
)

// RunIndex tries to return an index.gop and a http status code
func RunIndex(urlPath, parameters string) (string, string, int, string) {

	sourcePath := config.DocumentRoot + "/" + urlPath

	if !filesystem.Exists(sourcePath + "/" + "index.gop") { // If no index
		if filesystem.DirectoryExists(sourcePath) { // And its a directory
			return RunGop(config.DirectoryListingScript, parameters)
		}
	}

	return RunGop(urlPath+"/index.gop", parameters)
}
