package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultAutofillSearchLimit = 30
	defaultSearchResultLimit   = 30
)

type Settings struct {
	HideLabRooms        bool `json:"hide_lab_rooms"`
	AutofillSearchLimit int  `json:"autofill_search_limit"`
	SearchResultLimit   int  `json:"search_result_limit"`
}

func (s Settings) autofillSearchLimit() int {
	if s.AutofillSearchLimit <= 0 {
		return defaultAutofillSearchLimit
	}
	return s.AutofillSearchLimit
}

func (s Settings) searchResultLimit() int {
	if s.SearchResultLimit <= 0 {
		return defaultSearchResultLimit
	}
	return s.SearchResultLimit
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
		printText(fmt.Sprintf("2. Autofill Search Limit: %d", s.autofillSearchLimit()))
		printText(fmt.Sprintf("3. Search Result Limit: %d", s.searchResultLimit()))
		printText("0. Back")
		fmt.Println()
		line, back := readInputLine("Enter your choice (0-3): ")
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
		case "2":
			limit, ok := promptPositiveLimit("Autofill search limit", s.autofillSearchLimit())
			if !ok {
				continue
			}
			s.AutofillSearchLimit = limit
			if err := saveSettings(s); err != nil {
				errf("Could not save settings: %v\n", err)
				continue
			}
			okStyle.Printf("%sAutofill search limit set to %d.\n", indent, limit)
		case "3":
			limit, ok := promptPositiveLimit("Search result limit", s.searchResultLimit())
			if !ok {
				continue
			}
			s.SearchResultLimit = limit
			if err := saveSettings(s); err != nil {
				errf("Could not save settings: %v\n", err)
				continue
			}
			okStyle.Printf("%sSearch result limit set to %d.\n", indent, limit)
		default:
			wrnf("Invalid choice %q, please enter 0, 1, 2, or 3.\n", line)
		}
	}
}

func promptPositiveLimit(name string, current int) (int, bool) {
	line, back := readInputLine(fmt.Sprintf("%s (positive number, current %d): ", name, current))
	if back || strings.TrimSpace(line) == "" {
		return 0, false
	}
	limit, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || limit <= 0 {
		wrnf("Enter a positive whole number.\n")
		return 0, false
	}
	return limit, true
}

func labShownText(hidden bool) string {
	if hidden {
		return "will be"
	}
	return "will no longer be"
}
