package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Startup layer: exactly one term DB is searched at a time.
// One DB on disk implies itself (no prompt). Several DBs prompt once
// at startup; the choice can be changed later from the home menu.

// selectDatabaseAtStartup runs once before the home loop.
func selectDatabaseAtStartup() {
	files := listDBFiles()
	switch len(files) {
	case 0:
		return
	case 1:
		setActiveDB(files[0])
		printText(fmt.Sprintf("Term: %s", filepath.Base(files[0])))
		return
	}
	printH1("Select term database")
	printText("Multiple terms found — pick one to search.")
	printText("Switch anytime with 9 on the home screen.")
	promptSelectDatabase()
}

// promptSelectDatabase lists every term DB and records one choice.
// Back/0 keeps the current selection (if any). Reports success.
func promptSelectDatabase() bool {
	files := listDBFiles()
	sort.Strings(files)
	switch len(files) {
	case 0:
		wrnf("No databases found. Rebuild databases first (menu option 0).\n")
		return false
	case 1:
		setActiveDB(files[0])
		okStyle.Printf("%sTerm: %s\n", indent, filepath.Base(files[0]))
		return true
	}

	if cur := activeDBFile(); cur != "" {
		dimStyle.Printf("%sCurrent: %s\n", indent, filepath.Base(cur))
	}
	for i, f := range files {
		printText(fmt.Sprintf("  %d. %s", i+1, filepath.Base(f)))
	}
	for {
		line, back := readInputLine(fmt.Sprintf("Select term [1-%d, 0 = back]: ", len(files)))
		if back || strings.TrimSpace(line) == "0" {
			return activeDBFile() != ""
		}
		if n, err := strconv.Atoi(strings.TrimSpace(line)); err == nil && n >= 1 && n <= len(files) {
			setActiveDB(files[n-1])
			okStyle.Printf("%sTerm: %s\n", indent, filepath.Base(files[n-1]))
			return true
		}
		for _, f := range files {
			if strings.EqualFold(filepath.Base(f), strings.TrimSpace(line)) || strings.EqualFold(f, strings.TrimSpace(line)) {
				setActiveDB(f)
				okStyle.Printf("%sTerm: %s\n", indent, filepath.Base(f))
				return true
			}
		}
		wrnf("Unknown database %q.\n", line)
	}
}

// printActiveDB shows the one selected term on the home screen.
func printActiveDB() {
	if cur := activeDBFile(); cur != "" {
		printText(fmt.Sprintf("Term: %s", filepath.Base(cur)))
		return
	}
	if hasDatabases() {
		wrnf("No term selected.\n")
		return
	}
	printText("No databases yet")
}
