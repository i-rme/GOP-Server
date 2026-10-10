// Package controller turns a request for a script into the script's response.
package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	pages "github.com/i-rme/gop-server/src/handler/errors"
	"github.com/i-rme/gop-server/src/parser"
	"github.com/i-rme/gop-server/src/repository/cache"
	"github.com/i-rme/gop-server/src/repository/filesystem"
	"github.com/i-rme/gop-server/src/server/config"
	"github.com/i-rme/gop-server/src/worker"
)

// Response is what a script answered
type Response struct {
	Status int
	Header [][2]string // set by the script, in order
	Body   string
	Cache  string // X-Cache-Status: HIT when the script was already built
}

// Controller builds, caches and runs scripts
type Controller struct {
	config *config.Config
	log    *slog.Logger
	cache  *cache.Cache
	worker *worker.Worker
}

// New returns a Controller with an empty cache in CacheRoot
func New(c *config.Config, log *slog.Logger) (*Controller, error) {
	cacheRoot, err := filepath.Abs(c.CacheRoot)
	if err != nil {
		return nil, err
	}
	w, err := worker.New(worker.Options{
		CacheRoot:        cacheRoot,
		BuildTimeout:     c.BuildTimeout.Std(),
		ExecutionTimeout: c.ExecutionTimeout.Std(),
		AsNobody:         c.RunScriptsAsNobody,
	})
	if err != nil {
		return nil, err
	}
	return &Controller{config: c, log: log, cache: cache.New(cacheRoot), worker: w}, nil
}

// RunGop runs a .gop script
func (c *Controller) RunGop(urlPath, parameters string) Response {
	path := filepath.Join(c.config.DocumentRoot, urlPath)
	if !filesystem.Exists(path) {
		return c.fail(http.StatusNotFound, pages.NotFound, urlPath)
	}
	program, err := parser.Process(path, parser.Options{
		MySQL:             c.config.MySQLSupportEnabled,
		SessionCookieName: c.config.SessionCookieName,
		HeadersSeparator:  c.config.GopHeadersSeparator,
	})
	if err != nil {
		return c.fail(http.StatusInternalServerError, pages.BuildFailedParsing, err.Error())
	}
	return c.run(urlPath, program, parameters, true)
}

// RunGo runs a standalone .go program
func (c *Controller) RunGo(urlPath, parameters string) Response {
	path := filepath.Join(c.config.DocumentRoot, urlPath)
	src, err := os.ReadFile(path)
	if err != nil {
		return c.fail(http.StatusNotFound, pages.NotFound, urlPath)
	}
	return c.run(urlPath, parser.Plain(path, src), parameters, false)
}

// RunIndex runs a directory's index.gop, or the directory listing script when it has none
func (c *Controller) RunIndex(urlPath, parameters string) Response {
	directory := filepath.Join(c.config.DocumentRoot, urlPath)
	if !filesystem.Exists(filepath.Join(directory, "index.gop")) && filesystem.DirectoryExists(directory) {
		return c.RunGop(c.config.DirectoryListingScript, parameters)
	}
	return c.RunGop(urlPath+"/index.gop", parameters)
}

// run builds program unless it is cached, then runs it; gop selects the .gop pages and trailer
func (c *Controller) run(urlPath string, program *parser.Program, parameters string, gop bool) Response {
	binary, hit, err := c.build(program)
	switch {
	case errors.Is(err, worker.ErrTimeout):
		return c.fail(http.StatusInternalServerError, pages.BuildTimeout, urlPath)
	case err != nil:
		var build *worker.BuildError
		if !errors.As(err, &build) {
			return c.fail(http.StatusInternalServerError, pages.BuildFailedGopCompilationFatal, err.Error())
		}
		if gop {
			return c.fail(http.StatusInternalServerError, pages.BuildFailedGopCompilationSource, build.Output)
		}
		return c.fail(http.StatusInternalServerError, pages.BuildFailedGoCompilation, build.Output)
	}

	result, err := c.worker.Run(binary, parameters)
	var exit *exec.ExitError
	switch {
	case errors.Is(err, worker.ErrTimeout):
		return c.fail(http.StatusInternalServerError, pages.ExecutionTimeout, urlPath)
	case errors.As(err, &exit):
		page := pages.RunFailedGoFatal
		if gop {
			page = pages.RunFailedGopFatal
		}
		return c.fail(http.StatusInternalServerError, page, err.Error()+"\n"+result.Stdout+result.Stderr)
	case err != nil:
		return c.fail(http.StatusInternalServerError, pages.RunFailedGoFatal, err.Error())
	}
	if result.Stderr != "" {
		c.log.Debug("script wrote to stderr", "script", urlPath, "stderr", result.Stderr)
	}

	response := Response{Status: http.StatusOK, Body: result.Stdout, Cache: "MISS"}
	if hit {
		response.Cache = "HIT"
	}
	if !gop {
		response.Body = strings.TrimSuffix(response.Body, "\n")
		return response
	}
	body, trailer := parser.SplitOutput(result.Stdout, c.config.GopHeadersSeparator)
	response.Body = body
	if trailer == nil {
		c.log.Warn("script ended without its headers", "script", urlPath)
		return response
	}
	if trailer.Status != 0 {
		response.Status = trailer.Status
	}
	response.Header = trailer.Headers
	if !slices.ContainsFunc(response.Header, func(h [2]string) bool { return strings.EqualFold(h[0], "Content-Type") }) {
		response.Header = append(response.Header, [2]string{"Content-Type", "text/html"})
	}
	return response
}

// build returns the binary of program, building it unless it is cached; hit reports whether it was
func (c *Controller) build(program *parser.Program) (binary string, hit bool, err error) {
	key := cache.Key(program.Source)
	return c.cache.Get(key, func(dir, binary string) error {
		c.log.Debug("building", "script", program.Files[0], "key", key)
		files, main := program.Module()
		return c.worker.Build(dir, binary, files, main)
	})
}

// fail logs a failure and answers with its error page
func (c *Controller) fail(status, page int, details string) Response {
	c.log.Error("request failed", "status", status, "details", details)
	return Response{Status: status, Body: pages.Render(c.config.ServerSignature, page, details), Cache: "MISS"}
}
