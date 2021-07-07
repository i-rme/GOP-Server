package controller

import (
	"net/http"
	"pfg/src/handler/errors"
	"pfg/src/repository/cache"
	"pfg/src/repository/filesystem"
	"pfg/src/server/config"
	"pfg/src/server/logs"
	"pfg/src/worker"
)

// RunGo builds and returns a .go file and a http status code
func RunGo(urlPath, parameters string) (string, int, string) {
	sourcePath := config.DocumentRoot + "/" + urlPath

	if !filesystem.Exists(sourcePath) {
		return errors.Render(errors.NotFound, urlPath), http.StatusNotFound, "MISS"
	}

	cacheFile := cache.GetIfFresh(sourcePath)

	if cacheFile == "" { //File not cached
		logs.WriteDebug("The file is not on the cache. Building...")

		buildOutput, workingDirectory, buildOk := worker.BuildAndRun(sourcePath, parameters)

		if !buildOk {
			return errors.Render(errors.BuildFailedGoCompilation, buildOutput), http.StatusInternalServerError, "MISS"
		}

		cache.Put(sourcePath, workingDirectory, sourcePath)

		return buildOutput, http.StatusOK, "MISS"

	}

	//File cached
	output, runOk := worker.Run(cacheFile, parameters, 6)

	if !runOk {
		return errors.Render(errors.RunFailedGoFatal, output), http.StatusInternalServerError, "MISS"
	}

	return output, http.StatusOK, "HIT"

}
