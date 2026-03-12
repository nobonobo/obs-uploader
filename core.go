package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/events"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type AppDetection struct {
	Detected bool
	Process  *Process
}

type Core struct {
	service *AppService
	index   int
	tray    *application.SystemTray
}

func New() *Core {
	return &Core{
		service: &AppService{windows: map[string]*Window{}},
	}
}

func (c *Core) SetTray(tray *application.SystemTray) {
	c.tray = tray
	c.tray.SetIcon(trayIcons["light"])
	c.tray.SetDarkModeIcon(trayIcons["dark"])
}

func (c *Core) Service() *AppService {
	return c.service
}

func (c *Core) Run(ctx context.Context) error {
	states := map[string]State{}
	args := []goobs.Option{}
	if config.OBSPassword != "" {
		args = append(args, goobs.WithPassword(config.OBSPassword))
	}
	client, err := goobs.New(config.OBSAddress, args...)
	if err != nil {
		return err
	}
	defer client.Disconnect()

	version, err := client.General.GetVersion()
	if err != nil {
		return err
	}
	slog.Info("Connected to OBS", "version", version.ObsVersion)
	defer slog.Info("Disconnected from OBS")

	go func() {
		var lastDetected *Process
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-ctx.Done():
				client.Disconnect()
				return
			case <-ticker.C:
				var app *Process
				if lastDetected == nil {
					a, err := GetFullscreenApp()
					if err != nil {
						app = nil
					} else {
						app = a
					}
				} else {
					if ok, err := lastDetected.IsProcessExited(); err != nil {
						slog.Error("check process exit failed", "error", err)
					} else if ok {
						app = nil
					} else {
						app = lastDetected
					}
				}
				var event *AppDetection
				switch {
				case lastDetected != nil && app == nil:
					event = &AppDetection{
						Detected: false,
						Process:  lastDetected,
					}
				case lastDetected == nil && app != nil:
					event = &AppDetection{
						Detected: true,
						Process:  app,
					}
				}
				if event != nil {
					select {
					case client.IncomingEvents <- event:
					default:
					}
					lastDetected = app
				}
			}
		}
	}()
	var currentProcess *Process
	client.Listen(func(event any) {
		switch e := event.(type) {
		case *events.ScreenshotSaved:
			slog.Debug("screenshot saved", "event", e)
		case *events.RecordStateChanged:
			if currentProcess == nil {
				break
			}
			//log.Printf("record state changed: %v\n", e)
			state := states[currentProcess.Name]
			switch e.OutputState {
			case "OBS_WEBSOCKET_OUTPUT_STARTED":
				state.OutputPath = e.OutputPath
				state.Start = time.Now()
				slog.Info("record started", "path", e.OutputPath)
				c.tray.SetIcon(trayIcons["record"])
				c.tray.SetDarkModeIcon(trayIcons["record"])
			case "OBS_WEBSOCKET_OUTPUT_STOPPED":
				duration := time.Since(state.Start)
				slog.Info("record stopped", "path", e.OutputPath, "duration", duration)
				c.tray.SetIcon(trayIcons["light"])
				c.tray.SetDarkModeIcon(trayIcons["dark"])
				if state.OutputPath != e.OutputPath {
					break
				}
				prof, ok := config.Profiles[currentProcess.Name]
				if !ok {
					defaultProf, ok := config.Profiles["default"]
					prof = new(Profile)
					if ok {
						slog.Debug("using default profile", "profile", defaultProf)
						prof.Name = "default"
						prof.MinDuration = defaultProf.MinDuration
						prof.Template = defaultProf.Template
						for _, v := range defaultProf.Fields {
							prof.Fields = append(prof.Fields, v)
						}
					}
					prof.Name = currentProcess.Name
					for _, v := range prof.Fields {
						if v.ID == "Title" {
							v.Default = currentProcess.Name
						}
					}
				}
				if duration < time.Duration(prof.MinDuration*float64(time.Second)) {
					slog.Info("record duration is less than min duration, skipping upload", "duration", duration, "min", prof.MinDuration)
					break
				}
				if err := c.OpenWindow(prof, e.OutputPath); err != nil {
					slog.Error("failed to open window", "error", err)
				}
			}
			states[currentProcess.Name] = state
		case *AppDetection:
			if e.Detected {
				slog.Info("detected app", "name", e.Process.Name)
				currentProcess = e.Process
			} else {
				slog.Info("terminated app", "name", e.Process.Name)
				//currentProcess = nil
			}
		case *events.ExitStarted:
			slog.Info("Exit", "event", e)
		default:
			slog.Debug("unhandled event", "type", fmt.Sprintf("%T", event))
		}
	})
	return nil
}

func (c *Core) OpenWindow(profile *Profile, output string) error {
	c.index++
	slog.Info("opening window", "index", c.index, "profile", profile.Name, "output", output)
	c.service.openWindow(strconv.Itoa(c.index), *profile, output)
	return nil
}
