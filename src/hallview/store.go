package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

var dayNames = map[string]string{
	"2": "Monday",
	"3": "Tuesday",
	"4": "Wednesday",
	"5": "Thursday",
	"6": "Friday",
}

var dayOrder = []string{"2", "3", "4", "5", "6"}

type entry struct {
	building string
	room     string
	start    int
	end      int
	day      string
	course   string
	section  string
}

// Shared schedule store. Both Search by Room and Search by Time read
// through here so SQL, dedupe, and sort order stay identical.

func listDBFiles() []string {
	// DBs always live at <repo>/src/hallview/data/db, anchored by
	// repoRoot() so the binary works from any CWD.
	if files, _ := filepath.Glob(filepath.Join(dbDir(), "*.db")); len(files) > 0 {
		return files
	}
	return nil
}

func entryKey(e entry) string {
	return fmt.Sprintf("%s|%s|%d|%d|%s|%s|%s",
		e.building, e.room, e.start, e.end, e.day, e.course, e.section)
}

func dayRank(day string) int {
	switch day {
	case "2":
		return 0
	case "3":
		return 1
	case "4":
		return 2
	case "5":
		return 3
	case "6":
		return 4
	}
	return 99
}

func sortEntries(es []entry) {
	sort.Slice(es, func(i, j int) bool {
		ri, rj := dayRank(es[i].day), dayRank(es[j].day)
		if ri != rj {
			return ri < rj
		}
		if es[i].start != es[j].start {
			return es[i].start < es[j].start
		}
		if es[i].room != es[j].room {
			return es[i].room < es[j].room
		}
		return es[i].course < es[j].course
	})
}

// querySchedule scans every term DB with the same WHERE clause,
// dedupes identical rows, and returns per-term source names that hit.
func querySchedule(where string, args ...any) ([]entry, []string) {
	dbFiles := listDBFiles()
	seen := make(map[string]bool)
	var out []entry
	var sources []string
	for _, dbFile := range dbFiles {
		db, err := sql.Open("sqlite", dbFile)
		if err != nil {
			errf("Failed to open %s: %v\n", dbFile, err)
			continue
		}
		q := "SELECT building, room, start, end, day, course, section FROM schedule"
		if strings.TrimSpace(where) != "" {
			q += " WHERE " + where
		}
		rows, err := db.Query(q, args...)
		if err != nil {
			errf("Query failed in %s: %v\n", dbFile, err)
			db.Close()
			continue
		}
		count := 0
		for rows.Next() {
			var e entry
			if err := rows.Scan(&e.building, &e.room, &e.start, &e.end, &e.day, &e.course, &e.section); err != nil {
				continue
			}
			count++
			key := entryKey(e)
			if !seen[key] {
				seen[key] = true
				out = append(out, e)
			}
		}
		rows.Close()
		db.Close()
		logf("%s -> %d rows\n", dbFile, count)
		if count > 0 {
			sources = append(sources, filepath.Base(dbFile))
		}
	}
	sortEntries(out)
	return out, sources
}

func queryRoom(room string) ([]entry, []string) {
	return querySchedule("room = ? ORDER BY day, start", room)
}

func collectRooms(dbFiles []string) map[string]bool {
	allRooms := make(map[string]bool)
	for _, dbFile := range dbFiles {
		db, err := sql.Open("sqlite", dbFile)
		if err != nil {
			continue
		}
		rows, err := db.Query("SELECT DISTINCT room FROM schedule")
		if err != nil {
			db.Close()
			continue
		}
		for rows.Next() {
			var r string
			if rows.Scan(&r) == nil && r != "" {
				allRooms[r] = true
			}
		}
		rows.Close()
		db.Close()
	}
	return allRooms
}

// roomDayIndex groups entries as room -> day -> classes, plus a
// room -> building lookup. Time search works off this index so each
// room's overlap check is independent of DB layout.
func roomDayIndex(entries []entry) (map[string]map[string][]entry, map[string]string) {
	index := make(map[string]map[string][]entry)
	buildingOf := make(map[string]string)
	for _, e := range entries {
		if _, ok := buildingOf[e.room]; !ok {
			buildingOf[e.room] = e.building
		}
		if index[e.room] == nil {
			index[e.room] = make(map[string][]entry)
		}
		index[e.room][e.day] = append(index[e.room][e.day], e)
	}
	return index, buildingOf
}

// overlaps reports whether half-open [aStart,aEnd) hits [bStart,bEnd).
func overlaps(aStart, aEnd, bStart, bEnd int) bool {
	return aStart < bEnd && bStart < aEnd
}

func roomFreeOn(classes []entry, start, end int) bool {
	for _, e := range classes {
		if overlaps(e.start, e.end, start, end) {
			return false
		}
	}
	return true
}

// freeUntil returns the start of the next class at/after windowEnd,
// or 1440 when the room is free for the rest of the day.
func freeUntil(classes []entry, windowEnd int) int {
	next := 1440
	for _, e := range classes {
		if e.start >= windowEnd && e.start < next {
			next = e.start
		}
	}
	return next
}

// nextAfter returns the earliest class starting at/after windowEnd.
func nextAfter(classes []entry, windowEnd int) *entry {
	var best *entry
	for i := range classes {
		if classes[i].start < windowEnd {
			continue
		}
		if best == nil || classes[i].start < best.start {
			best = &classes[i]
		}
	}
	return best
}
