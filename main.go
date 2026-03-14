package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	core := New()
	app := application.New(application.Options{
		Name:        "obs-uploader",
		Description: "A demo of using raw HTML & CSS",
		Services: []application.Service{
			application.NewService(core.Service()),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Logger: config.Logger,
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		URL:    "dummy.html",
		Hidden: true,
	})

	go func() {
		defer slog.Info("terminated")
		for {
			if err := core.Run(ctx); err != nil {
				slog.Error("core.Run", "error", err)
			}
			if ctx.Err() != nil {
				break
			}
			time.Sleep(10 * time.Second)
		}
	}()

	trayMenu := app.NewMenu()
	trayMenu.Add("Quit").OnClick(func(ctx *application.Context) {
		slog.Info("Quit clicked")
		app.Quit()
	})
	systemTray := app.SystemTray.New()
	systemTray.SetMenu(trayMenu)
	core.SetTray(systemTray)

	// Start the application
	if err := app.Run(); err != nil {
		slog.Error("app.Run", "error", err)
		os.Exit(1)
	}
}
