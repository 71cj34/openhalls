package main

import (
	"fmt"
	"sort"
	"strings"

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

	for {
		fmt.Println()
		input, back := readInputLine("Room name or keyword (empty = back): ")
		if back || strings.TrimSpace(input) == "" {
			return
		}

		room, ok := fuzzyPick("room", keysAsCandidates(allRooms), input)
		if !ok {
			continue
		}

		results, sources := queryRoom(room)
		if len(results) == 0 {
			wrnf("No classes found for %s.\n", room)
			continue
		}

		printRoomReport(room, results, sources)
		headers, rows := entryRows(results)
		offerExportRows("room", room, headers, rows)
	}
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
	stdinReader.ReadString('\n')
}

// todo: get rid of this
func handle4() {
	runCustomQuery()
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
	fmt.Println()
	printText("A program made by Jason Cheng with ♡.")
	fmt.Println()
	printText("Repository: https://github.com/71cj34/openhalls")
	printText("Website: https://jasoncheng.me")
	fmt.Println()

	input, _ := readInputLine("Press enter to return: ")
	if input != "" {
		return
	}
}
