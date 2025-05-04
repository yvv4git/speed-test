package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/yvv4git/speed-test/internal/utils"
	"github.com/yvv4git/speed-test/internal/webclient"
	"github.com/yvv4git/speed-test/internal/webserver"
)

type ApplicationType string

const (
	ApplicationTypeServer ApplicationType = "server"
	ApplicationTypeClient ApplicationType = "client"
)

func main() {
	app := kingpin.New("websock-tunnel", "A tool for tunneling TCP over WebSocket.")
	appType := app.Flag("type", "Type of application to run (server or client).").Short('t').Required().Enum("server", "client")
	kingpin.MustParse(app.Parse(os.Args[1:]))

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("Starting application", "type", *appType)

	var err error
	switch ApplicationType(utils.Deref(appType)) {
	case ApplicationTypeServer:
		err = webserver.NewApplication(logger).Start(context.TODO())
	case ApplicationTypeClient:
		err = webclient.NewApplication(logger).Start(context.TODO())
	default:
		logger.Error("Unknown application type", "type", *appType)
		os.Exit(1)
	}

	if err != nil {
		logger.Error("Failed to start application", "error", err)
		os.Exit(1)
	}
}
