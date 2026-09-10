package main

import (
	"bufio"
	"fmt"
	"sort"
	"strings"
	"os"

	"github.com/fatih/color"
)

func handle1() {
	printH1("Search by Room")

	dbFiles := listDBFiles()
	if len(dbFiles) == 0 {
		wrnf("No databases found. Rebuild databases first (menu option 0).\n")
		pause()
		return
	}

	allRooms := collectRooms(dbFiles)
	if len(allRooms) == 0 {
		wrnf("Databases contain no rooms.\n")
		pause()
		return
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println()
		printTextf("Room name or keyword (empty = back): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			return
		}

		room, ok := pickRoom(reader, allRooms, input)
		if !ok {
			continue
		}

		results, sources := queryRoom(room)
		if len(results) == 0 {
			wrnf("No classes found for %s.\n", room)
			continue
		}

		printRoomReport(room, results, sources)
	}
}

// pickRoom resolves input to one room. Exact match wins; otherwise it
// shows a short numbered shortlist instead of silently guessing.
func pickRoom(reader *bufio.Reader, allRooms map[string]bool, input string) (string, bool) {
	for r := range allRooms {
		if strings.EqualFold(r, input) {
			return r, true
		}
	}

	lowerInput := strings.ToLower(input)
	type candidate struct {
		room  string
		score int
	}
	var candidates []candidate
	for r := range allRooms {
		lowerRoom := strings.ToLower(r)
		if !strings.Contains(lowerRoom, lowerInput) {
			continue
		}
		score := 100
		if strings.HasPrefix(lowerRoom, lowerInput) {
			score += 50
		}
		score += max(0, 20-len(r))
		candidates = append(candidates, candidate{room: r, score: score})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].room < candidates[j].room
	})

	if len(candidates) == 0 {
		wrnf("No rooms matching %q.\n", input)
		return "", false
	}

	limit := min(5, len(candidates))
	printH2(fmt.Sprintf("%d match(es), showing %d", len(candidates), limit))
	rows := make([][]string, 0, limit)
	for i := 0; i < limit; i++ {
		rows = append(rows, []string{fmt.Sprintf("%d", i+1), candidates[i].room})
	}
	printTable([]string{"#", "Room"}, rows)
	printTextf("Pick 1-%d [1], or type to refine: ", limit)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return candidates[0].room, true
	}
	var n int
	if _, err := fmt.Sscanf(line, "%d", &n); err == nil && n >= 1 && n <= limit {
		return candidates[n-1].room, true
	}
	return pickRoom(reader, allRooms, line)
}

// printRoomReport renders one compact table per day plus a free-time
// summary. One row per class replaces the old 5-line card.
func printRoomReport(room string, results []entry, sources []string) {
	building := results[0].building
	printH2(fmt.Sprintf("%s · %s — %d class(es)", room, building, len(results)))
	if len(sources) > 0 {
		dimStyle.Printf("%sTerms: %s\n", indent, strings.Join(sources, ", "))
	}

	byDay := make(map[string][]entry)
	for _, e := range results {
		byDay[e.day] = append(byDay[e.day], e)
	}

	for _, day := range dayOrder {
		classes := byDay[day]
		label := dayNames[day]
		if label == "" {
			label = "Day " + day
		}
		if len(classes) == 0 {
			fmt.Println()
			okStyle.Printf("%s%s — no classes, free all day\n", indent, label)
			continue
		}
		summary := summarizeFree(classes)
		fmt.Println()
		sectionStyle.Printf("%s%s (%d)\n", indent, label, len(classes))
		rows := make([][]string, 0, len(classes))
		for _, e := range classes {
			rows = append(rows, []string{
				fmt.Sprintf("%s–%s", fmtClock(e.start), fmtClock(e.end)),
				fmtDur(e.end - e.start),
				e.course,
				e.section,
			})
		}
		printTable([]string{"Time", "Dur", "Course", "Section"}, rows)
		freeStyle(summary.hasFree).Printf("%sFree: %s\n", indent, summary.text)
	}
	fmt.Println()
}

type freeSummary struct {
	text    string
	hasFree bool
}

var freeLineGreen = color.New(color.FgGreen)
var freeLineDim = color.New(color.Faint)

func freeStyle(hasFree bool) *color.Color {
	if hasFree {
		return freeLineGreen
	}
	return freeLineDim
}

// summarizeFree merges occupied intervals and lists usable gaps.
// Gaps under 15 minutes (passing periods) are folded into the count
// instead of printed, which keeps busy days to one line.
func summarizeFree(classes []entry) freeSummary {
	intervals := make([][2]int, 0, len(classes))
	for _, e := range classes {
		intervals = append(intervals, [2]int{e.start, e.end})
	}
	sort.Slice(intervals, func(i, j int) bool { return intervals[i][0] < intervals[j][0] })
	merged := [][2]int{}
	for _, iv := range intervals {
		if len(merged) > 0 && iv[0] <= merged[len(merged)-1][1] {
			if iv[1] > merged[len(merged)-1][1] {
				merged[len(merged)-1][1] = iv[1]
			}
		} else {
			merged = append(merged, iv)
		}
	}

	free := [][2]int{}
	cursor := 0
	for _, iv := range merged {
		if iv[0] > cursor {
			free = append(free, [2]int{cursor, iv[0]})
		}
		if iv[1] > cursor {
			cursor = iv[1]
		}
	}
	if cursor < 1440 {
		free = append(free, [2]int{cursor, 1440})
	}

	const dayStart, dayEnd = 8 * 60, 22 * 60
	parts := []string{}
	skipped := 0
	allDay := true
	for _, iv := range free {
		s, e := max(iv[0], 0), min(iv[1], 1440)
		if e-s < 15 {
			skipped++
			continue
		}
		allDay = allDay && s <= dayStart && e >= dayEnd
		var p string
		switch {
		case s <= 0 && e >= 1440:
			p = "all day"
		case s <= 0:
			p = fmt.Sprintf("until %s", fmtClock(e))
		case e >= 1440:
			p = fmt.Sprintf("after %s", fmtClock(s))
		default:
			p = fmt.Sprintf("%s–%s (%s)", fmtClock(s), fmtClock(e), fmtDur(e-s))
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		return freeSummary{text: "none today", hasFree: false}
	}
	text := strings.Join(parts, "; ")
	if skipped > 0 {
		text += fmt.Sprintf(" (+%d short gap(s) <15m)", skipped)
	}
	_ = allDay
	return freeSummary{text: text, hasFree: true}
}

func pause() {
	fmt.Println()
	printText("Press [Enter] to continue...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func handle3() {
	printH1("Search by Course")
	wrnf("Not implemented yet.\n")
	pause()
}
func handle4() {
	printH1("Custom SQL Query")
	wrnf("Not implemented yet.\n")
	pause()
}
func handle5() {
	printH1("Manage Favorites")
	wrnf("Not implemented yet.\n")
	pause()
}
func handle6() {
	printH1("Settings")
	wrnf("Not implemented yet.\n")
	pause()
}
func handle7() {
	printH1("About")
	printText("Hallview — find free rooms and lecture times.")
	pause()
}
