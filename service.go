package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type Info struct {
	Name       string   `json:"name"`
	OutputPath string   `json:"outputPath"`
	Fields     FieldDef `json:"fields"`
}

type Window struct {
	window *application.WebviewWindow
	Info
}

type AppService struct {
	mu      sync.RWMutex
	windows map[string]*Window
}

func (s *AppService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	log.Println("OnStartup")
	if err := LoadConfig(); err != nil {
		app := application.Get()
		time.Sleep(500 * time.Millisecond)
		errDialog := app.Dialog.Error().SetTitle("Configuration Error").SetMessage(err.Error())
		errDialog.Show()
		os.Exit(1)
	}
	return nil
}

// SetWindow allows passing the main application window so the service can interact with it
func (s *AppService) openWindow(id string, prof Profile, output string) {
	app := application.Get()
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "OBS Recorder",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/?id=" + id,
	})
	window.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.windows, id)
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	s.windows[id] = &Window{
		window: window,
		Info: Info{
			Name:       prof.Name,
			OutputPath: output,
			Fields:     prof.Fields,
		},
	}
}

// GetFieldDef returns the expected dynamic form fields.
func (s *AppService) GetInfo(id string) Info {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.windows[id].Info
}

// Upload receives the dynamic form data and processes it.
func (s *AppService) Upload(id string, data map[string]any) {
	template := "default.json"
	info, ok := s.windows[id]
	if ok {
		if prof, ok := config.Profiles[info.Name]; ok {
			template = prof.Template
		}
	}
	templatePath := filepath.Join(config.ProfileDir, template)
	log.Printf("Upload called with data: %+v\n", data)
	args := []string{"-t", templatePath, info.OutputPath}
	for k, v := range data {
		args = append(args, fmt.Sprintf("%s=%q", k, v))
	}
	cmd := exec.Command("thumb-tool",
		args...,
	)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		msg := fmt.Sprintf("Error: %s\n\n%s", err.Error(), strings.TrimSpace(string(output)))
		app := application.Get()
		errDialog := app.Dialog.Error().SetTitle("Upload Failed").SetMessage(msg)
		errDialog.Show()
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.windows[id].window != nil {
		s.windows[id].window.Close()
	}
	delete(s.windows, id)
}

// Cancel closes the window entirely instead of uploading.
func (s *AppService) Cancel(id string) {
	log.Println("Cancel called")
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.windows[id].window != nil {
		s.windows[id].window.Close()
	}
	delete(s.windows, id)
}
