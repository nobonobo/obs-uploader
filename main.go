package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[string]("time")
}

func main() {
	core := New()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	app := application.New(application.Options{
		Name:        "obs-recorder",
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
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		URL:    "dummy.html",
		Hidden: true,
	})

	go func() {
		defer log.Println("terminated")
		for {
			if err := core.Run(ctx); err != nil {
				log.Println(err)
			}
			if ctx.Err() != nil {
				break
			}
			time.Sleep(5 * time.Second)
		}
	}()

	trayMenu := app.NewMenu()
	trayMenu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	systemTray := app.SystemTray.New()
	systemTray.SetMenu(trayMenu)

	// Start the application
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
