package controller

import (
	"net/http"
	"path/filepath"

	"github.com/i-rme/gop-server/src/handler/errors"
	"github.com/i-rme/gop-server/src/repository/filesystem"
	"github.com/i-rme/gop-server/src/worker"
)

// RunGo builds and runs a .go file, returning its output, an HTTP status and the cache status
func (c *Controller) RunGo(urlPath, parameters string) (string, int, string) {
	sourcePath := filepath.Join(c.config.DocumentRoot, urlPath)

	if !filesystem.Exists(sourcePath) {
		return c.errorPage(errors.NotFound, urlPath), http.StatusNotFound, "MISS"
	}

	binary := c.cache.GetIfFresh(sourcePath)
	if binary == "" {
		c.log.Debug("building", "script", sourcePath)

		output, workDir, err := worker.BuildAndRun(sourcePath, parameters)
		if err != nil {
			return c.errorPage(errors.BuildFailedGoCompilation, output), http.StatusInternalServerError, "MISS"
		}
		c.cache.Put(sourcePath, workDir, sourcePath)
		return output, http.StatusOK, "MISS"
	}

	output, err := worker.Run(binary, parameters, executionTimeout, c.config.RunScriptsAsNobody)
	if err == worker.ErrTimeout {
		return c.errorPage(errors.ExecutionTimeout, binary), http.StatusInternalServerError, "MISS"
	}
	if err != nil {
		return c.errorPage(errors.RunFailedGoFatal, output), http.StatusInternalServerError, "MISS"
	}
	return output, http.StatusOK, "HIT"
}
