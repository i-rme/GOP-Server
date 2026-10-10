package controller

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/i-rme/gop-server/src/handler/errors"
	"github.com/i-rme/gop-server/src/parser"
	"github.com/i-rme/gop-server/src/repository/filesystem"
	"github.com/i-rme/gop-server/src/worker"
)

// RunGop builds and runs a .gop file, returning its output, the headers it set as JSON,
// an HTTP status and the cache status
func (c *Controller) RunGop(urlPath, parameters string) (string, string, int, string) {
	sourcePath := filepath.Join(c.config.DocumentRoot, urlPath)

	if !filesystem.Exists(sourcePath) {
		return c.errorPage(errors.NotFound, urlPath), "[]", http.StatusNotFound, "MISS"
	}

	binary := c.cache.GetIfFresh(sourcePath)
	if binary == "" {
		c.log.Debug("building", "script", sourcePath)

		program, err := parser.Process(sourcePath, parser.Options{
			MySQL:             c.config.MySQLSupportEnabled,
			SessionCookieName: c.config.SessionCookieName,
			HeadersSeparator:  c.config.GopHeadersSeparator,
		})
		if err != nil {
			return c.errorPage(errors.BuildFailedParsing, err.Error()), "[]", http.StatusInternalServerError, "MISS"
		}
		tempPath, err := filesystem.WriteTemp(c.config.CacheRoot, "temp-*.go", program.Source)
		if err != nil {
			return c.errorPage(errors.BuildFailedGopCompilationFatal, err.Error()), "[]", http.StatusInternalServerError, "MISS"
		}
		defer os.Remove(tempPath)

		combined, workDir, err := worker.BuildAndRun(tempPath, parameters)
		if err != nil {
			if workDir == "" {
				return c.errorPage(errors.BuildFailedGopCompilationFatal, combined), "[]", http.StatusInternalServerError, "MISS"
			}
			return c.errorPage(errors.BuildFailedGopCompilationSource, combined), "[]", http.StatusInternalServerError, "MISS"
		}
		c.cache.Put(sourcePath, workDir, tempPath)

		output, headers := parser.SplitOutputAndHeaders(combined, c.config.GopHeadersSeparator)
		return output, headers, http.StatusOK, "MISS"
	}

	combined, err := worker.Run(binary, parameters, executionTimeout, c.config.RunScriptsAsNobody)
	if err == worker.ErrTimeout {
		return c.errorPage(errors.ExecutionTimeout, binary), "[]", http.StatusInternalServerError, "MISS"
	}
	if err != nil {
		return c.errorPage(errors.RunFailedGopFatal, combined), "[]", http.StatusInternalServerError, "MISS"
	}
	output, headers := parser.SplitOutputAndHeaders(combined, c.config.GopHeadersSeparator)
	return output, headers, http.StatusOK, "HIT"
}

// RunIndex runs a directory's index.gop, or the directory listing script when it has none
func (c *Controller) RunIndex(urlPath, parameters string) (string, string, int, string) {
	directory := filepath.Join(c.config.DocumentRoot, urlPath)

	if !filesystem.Exists(filepath.Join(directory, "index.gop")) && filesystem.DirectoryExists(directory) {
		return c.RunGop(c.config.DirectoryListingScript, parameters)
	}
	return c.RunGop(urlPath+"/index.gop", parameters)
}
