// Command gop-server serves .gop and .go scripts, and static files, over HTTP.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/i-rme/gop-server/src/server"
	"github.com/i-rme/gop-server/src/server/config"
	"github.com/i-rme/gop-server/src/server/logs"
)

func main() {
	configPath := flag.String("config", "config/configuration.json", "configuration file")
	flag.Parse()

	c, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	logger, logFile, err := logs.New(c.LogRoot, c.LoggingLevel)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	for _, warning := range c.Warnings {
		logger.Warn(warning)
	}

	srv, err := server.New(c, logger)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
