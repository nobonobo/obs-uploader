package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Field struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default any    `json:"default"`
}

type FieldDef []Field

type Profile struct {
	Name        string   `json:"name"`
	Template    string   `json:"template"`
	MinDuration float64  `json:"min-duration"`
	Fields      FieldDef `json:"fields"`
}

type State struct {
	Start      time.Time
	OutputPath string
}

var config = struct {
	OBSAddress  string
	OBSPassword string
	ProfileDir  string
	Profiles    map[string]*Profile
}{
	OBSAddress:  "localhost:4455",
	OBSPassword: "",
	ProfileDir:  "profiles",
	Profiles:    map[string]*Profile{},
}

func LoadConfig() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	name := strings.TrimSuffix(filepath.Base(executable), filepath.Ext(executable))
	configRoot, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get user config directory: %w", err)
	}
	config.ProfileDir = filepath.Join(configRoot, name)
	flag.StringVar(&config.OBSAddress, "address", config.OBSAddress, "OBS WebSocket address")
	flag.StringVar(&config.OBSPassword, "password", config.OBSPassword, "OBS WebSocket password")
	flag.StringVar(&config.ProfileDir, "profile", config.ProfileDir, "Profile directory")
	flag.Parse()
	if err := os.MkdirAll(config.ProfileDir, 0755); err != nil {
		log.Fatal(err)
	}
	if err := filepath.WalkDir(config.ProfileDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".profile" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read profile %s: %w", path, err)
		}
		var profile *Profile
		if err := json.Unmarshal(data, &profile); err != nil {
			return fmt.Errorf("failed to unmarshal profile %s: %w", path, err)
		}
		config.Profiles[profile.Name] = profile
		return nil
	}); err != nil {
		return fmt.Errorf("failed to load profiles: %w", err)
	}
	if len(config.Profiles) == 0 {
		return fmt.Errorf("no such profiles: you need profiles in %q", config.ProfileDir)
	}
	log.Println("Loaded profiles:", config.Profiles)
	return nil
}
