package main

import (
	"os"
	"path/filepath"
)

// we gotta do all this so that the executable works from every cwd
// probably isn't even needed......
func repoRoot() string {
	var candidates []string

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		if resolved, err := filepath.EvalSymlinks(exeDir); err == nil {
			exeDir = resolved
		}
		candidates = append(candidates,
			exeDir,
			filepath.Join(exeDir, ".."),
			filepath.Join(exeDir, "..", ".."),
			filepath.Join(exeDir, "..", "..", ".."),
		)
	}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			cwd,
			filepath.Join(cwd, ".."),
			filepath.Join(cwd, "..", ".."),
			filepath.Join(cwd, "..", "..", ".."),
		)
	}

	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if st, err := os.Stat(filepath.Join(abs, "schedules")); err == nil && st.IsDir() {
			return abs
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

func scheduleDir() string {
	return filepath.Join(repoRoot(), "schedules")
}

func coursesDir() string {
	return filepath.Join(repoRoot(), "courses")
}

func stateDir() string {
	return filepath.Join(repoRoot(), "state")
}

func xmlDir() string {
	return filepath.Join(repoRoot(), "xml")
}

func dbDir() string {
	return filepath.Join(repoRoot(), "src", "hallview", "data", "db")
}
