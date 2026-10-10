// Package controller turns a request for a script into the script's output.
package controller

import (
	"log/slog"
	"time"

	"github.com/i-rme/gop-server/src/handler/errors"
	"github.com/i-rme/gop-server/src/repository/cache"
	"github.com/i-rme/gop-server/src/server/config"
)

// executionTimeout bounds a cached script's run
const executionTimeout = 6 * time.Second

// Controller builds, caches and runs scripts
type Controller struct {
	config *config.Config
	log    *slog.Logger
	cache  *cache.Cache
}

// New returns a Controller with an empty cache
func New(c *config.Config, log *slog.Logger) *Controller {
	return &Controller{config: c, log: log, cache: cache.New()}
}

// errorPage logs a failure and renders its error page
func (c *Controller) errorPage(code int, details string) string {
	c.log.Error("request failed", "page", code, "details", details)
	return errors.Render(c.config.ServerSignature, code, details)
}
