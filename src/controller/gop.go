package controller

import (
	"net/http"
	"pfg/src/handler/errors"
	"pfg/src/parser"
	"pfg/src/repository/cache"
	"pfg/src/repository/filesystem"
	"pfg/src/server/config"
	"pfg/src/server/logs"
	"pfg/src/worker"
)

// RunGop builds and returns a .gop file and a http status code
func RunGop(urlPath, parameters string) (string, string, int, string) {
	sourcePath := config.DocumentRoot + "/" + urlPath

	if !filesystem.Exists(sourcePath) {
		return errors.Render(errors.NotFound, urlPath), "[]", http.StatusNotFound, "MISS"
	}

	cacheFile := cache.GetIfFresh(sourcePath)

	if cacheFile == "" { //File not cached
		logs.WriteDebug("The file is not on the cache. Building...")

		program, err := parser.Process(sourcePath, parser.Options{
			MySQL:             config.MySQLSupportEnabled,
			SessionCookieName: config.SessionCookieName,
			HeadersSeparator:  config.GopHeadersSeparator,
		})
		if err != nil {
			return errors.Render(errors.BuildFailedParsing, err.Error()), "[]", http.StatusInternalServerError, "MISS"
		}
		tempFilePath := filesystem.WriteTemp(config.CacheRoot, program.Source)

		combinedOutput, workingDirectory, buildOk := worker.BuildAndRun(tempFilePath, parameters)

		if !buildOk {

			if workingDirectory == "" {
				return errors.Render(errors.BuildFailedGopCompilationFatal, combinedOutput), "[]", http.StatusInternalServerError, "MISS"
			}

			return errors.Render(errors.BuildFailedGopCompilationSource, combinedOutput), "[]", http.StatusInternalServerError, "MISS"
		}

		cache.Put(sourcePath, workingDirectory, tempFilePath)
		filesystem.Remove(tempFilePath)

		buildOutput, headersOutput := parser.SplitOutputAndHeaders(combinedOutput, config.GopHeadersSeparator)

		return buildOutput, headersOutput, http.StatusOK, "MISS"

	}

	//File cached
	combinedOutput, runOk := worker.Run(cacheFile, parameters, 6)

	if !runOk {
		return errors.Render(errors.RunFailedGopFatal, combinedOutput), "[]", http.StatusInternalServerError, "MISS"
	}

	buildOutput, headersOutput := parser.SplitOutputAndHeaders(combinedOutput, config.GopHeadersSeparator)
	return buildOutput, headersOutput, http.StatusOK, "HIT"

}
