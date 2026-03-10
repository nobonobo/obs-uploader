package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
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

func init() {
	flag.StringVar(&config.OBSAddress, "address", config.OBSAddress, "OBS WebSocket address")
	flag.StringVar(&config.OBSPassword, "password", config.OBSPassword, "OBS WebSocket password")
	flag.StringVar(&config.ProfileDir, "profile", config.ProfileDir, "Profile directory")
	flag.Parse()
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
			return err
		}
		var profile *Profile
		if err := json.Unmarshal(data, &profile); err != nil {
			return err
		}
		config.Profiles[profile.Name] = profile
		return nil
	}); err != nil {
		log.Fatalln(err)
	}
	log.Println("Loaded profiles:", config.Profiles)
}
