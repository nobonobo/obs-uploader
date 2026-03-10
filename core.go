package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/events"
)

type AppDetection struct {
	Detected bool
	Process  *Process
}

type Core struct {
	service *AppService
	index   int
}

func New() *Core {
	return &Core{service: &AppService{windows: map[string]*Window{}}}
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
	log.Println("Connected to OBS Version:", version.ObsVersion)
	defer log.Println("Disconnected from OBS")

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
						log.Println(err)
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
	var currentProfile *Profile
	client.Listen(func(event any) {
		switch e := event.(type) {
		case *events.ScreenshotSaved:
			log.Printf("screenshot saved: %v\n", e)
		case *events.RecordStateChanged:
			if currentProcess == nil {
				break
			}
			//log.Printf("record state changed: %v\n", e)
			prof, ok := config.Profiles[currentProcess.Name]
			if !ok {
				prof = new(Profile)
				*prof = *config.Profiles["default"]
				prof.Name = currentProcess.Name
			}
			currentProfile = prof
			state := states[currentProcess.Name]
			switch e.OutputState {
			case "OBS_WEBSOCKET_OUTPUT_STARTED":
				state.OutputPath = e.OutputPath
				state.Start = time.Now()
				log.Println("record started:", e.OutputPath)
			case "OBS_WEBSOCKET_OUTPUT_STOPPED":
				if state.OutputPath != e.OutputPath {
					break
				}
				duration := time.Since(state.Start)
				log.Println("record stopped:", e.OutputPath, "duration:", duration, currentProfile)
				if duration < time.Duration(currentProfile.MinDuration*float64(time.Second)) {
					log.Println("record duration is less than min duration, skipping upload")
					break
				}
				if err := c.OpenWindow(currentProfile, e.OutputPath); err != nil {
					log.Println(err)
				}
			}
			states[currentProcess.Name] = state
		case *AppDetection:
			if e.Detected {
				log.Printf("detected app: %v\n", e.Process.Name)
				currentProcess = e.Process
			} else {
				log.Printf("terminated app: %v\n", e.Process.Name)
				currentProcess = nil
			}
		case *events.ExitStarted:
			log.Printf("Exit: %#v", e)
		default:
			log.Printf("unhandled: %T\n", event)
		}
	})
	return nil
}

func (c *Core) OpenWindow(profile *Profile, output string) error {
	c.index++
	log.Println("opening window:", c.index, profile, output)
	c.service.openWindow(strconv.Itoa(c.index), *profile, output)
	return nil
}
