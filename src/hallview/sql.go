package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func createSchedDB() {
	scheduleDir := scheduleDir()
	dbDir := dbDir()

	if err := os.MkdirAll(dbDir, 0755); err != nil {
		errf("Error creating db directory: %v\n", err)
		return
	}

	err := filepath.Walk(scheduleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".json") || strings.HasPrefix(info.Name(), "_") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			errf("Error reading %s: %v\n", path, err)
			return nil
		}

		var raw map[string]map[string][]struct {
			Start   float64 `json:"start"`
			End     float64 `json:"end"`
			Day   string  `json:"day"`
			Course  string  `json:"course"`
			Display string  `json:"display"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			errf("Error parsing %s: %v\n", path, err)
			return nil
		}

		dbPath := filepath.Join(dbDir, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))+".db")

		os.Remove(dbPath)
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			errf("Error opening db %s: %v\n", dbPath, err)
			return nil
		}
		defer db.Close()

		_, err = db.Exec(`CREATE TABLE schedule (
			building TEXT,
			room TEXT,
			start INTEGER,
			end INTEGER,
			day TEXT,
			course TEXT,
			section TEXT
		)`)
		if err != nil {
			errf("Error creating table in %s: %v\n", dbPath, err)
			return nil
		}

		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA synchronous=NORMAL")

		tx, err := db.Begin()
		if err != nil {
			errf("Error starting transaction for %s: %v\n", dbPath, err)
			return nil
		}

		stmt, err := tx.Prepare("INSERT INTO schedule (building, room, start, end, day, course, section) VALUES (?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			errf("Error preparing statement for %s: %v\n", dbPath, err)
			return nil
		}

		for b, rs := range raw {
			for r, entries := range rs {
				for _, entry := range entries {
					_, err := stmt.Exec(b, r, int(entry.Start), int(entry.End), entry.Day, entry.Course, entry.Display)
					if err != nil {
						errf("Error inserting into %s: %v\n", dbPath, err)
					}
				}
			}
		}

		stmt.Close()
		err = tx.Commit()
		if err != nil {
			errf("Error committing transaction for %s: %v\n", dbPath, err)
			return nil
		}

		logf("Created: %s\n", dbPath)
		return nil
	})

	if err != nil {
		errf("Error walking schedules: %v\n", err)
	}
}
