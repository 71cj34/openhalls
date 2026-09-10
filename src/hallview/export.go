package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// exportsDir returns the CSV output folder, <repo>/exports.
func exportsDir() string {
	return filepath.Join(repoRoot(), "exports")
}

func exportSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case r == ' ' || r == '-' || r == '_':
			out = append(out, '_')
		}
	}
	slug := strings.Trim(string(out), "_")
	for strings.Contains(slug, "__") {
		slug = strings.ReplaceAll(slug, "__", "_")
	}
	return slug
}

func exportFileName(kind, label string) string {
	slug := exportSlug(label)
	if slug == "" {
		slug = "results"
	}
	if len(slug) > 40 {
		slug = slug[:40]
	}
	stamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s-%s.csv", kind, slug, stamp)
}

func writeExportCSV(kind, label string, headers []string, rows [][]string) (string, error) {
	dir := exportsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, exportFileName(kind, label))
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write(headers); err != nil {
		return "", err
	}
	for _, r := range rows {
		rec := make([]string, len(headers))
		for i := range headers {
			if i < len(r) {
				rec[i] = r[i]
			}
		}
		if err := w.Write(rec); err != nil {
			return "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return path, nil
}

// offerExportRows prompts y/N and writes the full row set to exports/.
func offerExportRows(kind, label string, headers []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}
	fmt.Println()
	answer, back := readInputLine(fmt.Sprintf("Export %d row(s) to CSV [y/N]? ", len(rows)))
	if back || strings.TrimSpace(answer) == "" {
		return
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
	default:
		return
	}
	path, err := writeExportCSV(kind, label, headers, rows)
	if err != nil {
		errf("Export failed: %v\n", err)
		return
	}
	okStyle.Printf("%sSaved %d row(s) to %s\n", indent, len(rows), path)
}

// entryRows flattens entries to exportable rows with stable headers.
func entryRows(entries []entry) ([]string, [][]string) {
	headers := []string{"building", "room", "start", "end", "day", "course", "section"}
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []string{
			e.building,
			e.room,
			fmt.Sprintf("%d", e.start),
			fmt.Sprintf("%d", e.end),
			e.day,
			e.course,
			e.section,
		})
	}
	return headers, rows
}

// freeHitsToRows flattens free-room day hits for export.
func freeHitsToRows(days []string, dayHits map[string][]roomHit) ([]string, [][]string) {
	headers := []string{"day", "room", "building", "free_until", "next_class"}
	var rows [][]string
	for _, day := range days {
		label := dayNames[day]
		if label == "" {
			label = "Day " + day
		}
		for _, h := range dayHits[day] {
			next := ""
			if h.next != nil {
				next = fmt.Sprintf("%s (%s)", fmtClock(h.next.start), h.next.course)
			}
			rows = append(rows, []string{label, h.room, h.building, fmtClock(h.until), next})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i][0] != rows[j][0] {
			return rows[i][0] < rows[j][0]
		}
		return rows[i][1] < rows[j][1]
	})
	return headers, rows
}

// busyRowsToRows flattens busy-window matches for export.
func busyRowsToRows(days []string, dayRows map[string][][]string) ([]string, [][]string) {
	headers := []string{"day", "time", "room", "course", "section"}
	var rows [][]string
	for _, day := range days {
		label := dayNames[day]
		if label == "" {
			label = "Day " + day
		}
		for _, r := range dayRows[day] {
			rec := make([]string, 0, 5)
			rec = append(rec, label)
			for i := 0; i < 4; i++ {
				if i < len(r) {
					rec = append(rec, r[i])
				} else {
					rec = append(rec, "")
				}
			}
			rows = append(rows, rec)
		}
	}
	return headers, rows
}
