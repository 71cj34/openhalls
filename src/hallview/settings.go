package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Settings struct {
	HideLabRooms bool `json:"hide_lab_rooms"`
}

func settingsPath() string {
	return filepath.Join(repoRoot(), "src", "hallview", "data", "settings.json")
}

func loadSettings() Settings {
	s := Settings{}
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return s
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return s
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s
	}
	return s
}

func saveSettings(s Settings) error {
	if err := os.MkdirAll(filepath.Dir(settingsPath()), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath(), append(data, '\n'), 0644)
}

func settingsOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func hideLabRooms() bool {
	return loadSettings().HideLabRooms
}

func handle6() {
	printH1("Settings")
	for {
		s := loadSettings()
		fmt.Println()
		printH2("Options")
		printText(fmt.Sprintf("1. Hide Lab Rooms: %s", settingsOnOff(s.HideLabRooms)))
		printText("0. Back")
		fmt.Println()
		line, back := readInputLine("Enter your choice (0-1): ")
		if back || strings.TrimSpace(line) == "" {
			return
		}
		switch strings.TrimSpace(line) {
		case "0":
			return
		case "1":
			s.HideLabRooms = !s.HideLabRooms
			if err := saveSettings(s); err != nil {
				errf("Could not save settings: %v\n", err)
				continue
			}
			okStyle.Printf("%sHide Lab Rooms: %s. Lab rooms %s hidden in time-based searches.\n", indent, settingsOnOff(s.HideLabRooms), labShownText(s.HideLabRooms))
		default:
			wrnf("Invalid choice %q, please enter 0 or 1.\n", line)
		}
	}
}

func labShownText(hidden bool) string {
	if hidden {
		return "will be"
	}
	return "will no longer be"
}