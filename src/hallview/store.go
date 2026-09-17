package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
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
	isLab    bool
}

// SEE THE ACTIVEDB CODE...
var activeDB string

func setActiveDB(path string) {
	activeDB = path
}

func activeDBFile() string {
	if activeDB != "" {
		// the implication
		if st, err := os.Stat(activeDB); err == nil && !st.IsDir() {
			return activeDB
		}
		activeDB = ""
	}
	files := listDBFiles()
	if len(files) == 1 {
		activeDB = files[0]
		return activeDB
	}
	return ""
}
func listDBFiles() []string {
	// !make sure reporoot was initialized before it does this
	if files, _ := filepath.Glob(filepath.Join(dbDir(), "*.db")); len(files) > 0 {
		return files
	}
	return nil
}

func entryKey(e entry) string {
	return fmt.Sprintf("%s|%s|%d|%d|%s|%s|%s|%t",
		e.building, e.room, e.start, e.end, e.day, e.course, e.section, e.isLab)
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

// todo: dedupe
func querySchedule(where string, args ...any) ([]entry, []string) {
	dbFile := activeDBFile()
	if dbFile == "" {
		return nil, nil
	}
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		errf("Failed to open %s: %v\n", dbFile, err)
		return nil, nil
	}
	defer db.Close()

	// todo: deprecate section prefix scanner (will break stuff)
	q := "SELECT building, room, start, end, day, course, section, is_lab FROM schedule"
	if strings.TrimSpace(where) != "" {
		q += " WHERE " + where
	}
	withLab := true
	rows, err := db.Query(q, args...)
	if err != nil {
		q = "SELECT building, room, start, end, day, course, section FROM schedule"
		if strings.TrimSpace(where) != "" {
			q += " WHERE " + where
		}
		rows, err = db.Query(q, args...)
		if err != nil {
			errf("Query failed in %s: %v\n", dbFile, err)
			return nil, nil
		}
		withLab = false
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var out []entry
	count := 0
	for rows.Next() {
		var e entry
		var isLab int
		if withLab {
			err = rows.Scan(&e.building, &e.room, &e.start, &e.end, &e.day, &e.course, &e.section, &isLab)
			e.isLab = isLab != 0
		} else {
			err = rows.Scan(&e.building, &e.room, &e.start, &e.end, &e.day, &e.course, &e.section)
			e.isLab = isLabSection(e.section)
		}
		if err != nil {
			continue
		}
		count++
		key := entryKey(e)
		if !seen[key] {
			seen[key] = true
			out = append(out, e)
		}
	}
	logf("%s -> %d rows\n", dbFile, count)
	var sources []string
	if count > 0 {
		sources = append(sources, filepath.Base(dbFile))
	}
	sortEntries(out)
	return out, sources
}

func queryRoom(room string) ([]entry, []string) {
	return querySchedule("room = ? ORDER BY day, start", room)
}

func queryCourse(course string) ([]entry, []string) {
	return querySchedule("course = ? ORDER BY day, start", course)
}

func collectDistinct(column string) map[string]bool {
	out := make(map[string]bool)
	dbFile := activeDBFile()
	if dbFile == "" {
		return out
	}
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return out
	}
	rows, err := db.Query("SELECT DISTINCT " + column + " FROM schedule")
	if err != nil {
		db.Close()
		return out
	}
	for rows.Next() {
		var v string
		if rows.Scan(&v) == nil && v != "" {
			out[v] = true
		}
	}
	rows.Close()
	db.Close()
	return out
}

func collectRooms() map[string]bool {
	return collectDistinct("room")
}

// sends off lab rooms
func labRoomSet() map[string]bool {
	out := make(map[string]bool)
	dbFile := activeDBFile()
	if dbFile == "" {
		return out
	}
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return out
	}
	defer db.Close()
	rows, err := db.Query("SELECT DISTINCT room FROM schedule WHERE is_lab = 1")
	if err != nil {
		rows, err = db.Query("SELECT DISTINCT room FROM schedule WHERE upper(section) LIKE 'LAB%'")
		if err != nil {
			return out
		}
	}
	defer rows.Close()
	for rows.Next() {
		var r string
		if rows.Scan(&r) == nil && r != "" {
			out[r] = true
		}
	}
	return out
}

func collectCourses() map[string]bool {
	return collectDistinct("course")
}

// todo: normalize spaces
// todo: collapse searching all catalogs to a settings option
func courseTitles() map[string]string {
	titles := make(map[string]string)
	var files []string
	if cur := activeDBFile(); cur != "" {
		base := strings.TrimSuffix(filepath.Base(cur), filepath.Ext(cur))
		cand := filepath.Join(coursesDir(), base+".json")
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			files = []string{cand}
		}
	}
	if len(files) == 0 {
		pattern := filepath.Join(coursesDir(), "*.json")
		files, _ = filepath.Glob(pattern)
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var catalog []struct {
			Code  string `json:"code"`
			Title string `json:"title"`
		}
		if err := json.Unmarshal(data, &catalog); err != nil {
			continue
		}
		for _, c := range catalog {
			key := strings.ReplaceAll(strings.TrimSpace(c.Code), " ", "-")
			if key != "" && titles[key] == "" {
				titles[key] = strings.TrimSpace(c.Title)
			}
		}
	}
	return titles
}

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

// overlaps reports wether half-open [aStart,aEnd) hits [bStart,bEnd)
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

// todo: make the no more classes til end of day case not ugly
func freeUntil(classes []entry, windowEnd int) int {
	next := 1440
	for _, e := range classes {
		if e.start >= windowEnd && e.start < next {
			next = e.start
		}
	}
	return next
}

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
