package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"os"

	"github.com/fatih/color"
	_ "modernc.org/sqlite"
)

func handle1() {
	clearScreen()
	printH1("Search by Room")
	fmt.Println()

	// Collect all .db files
	dbFiles, err := filepath.Glob("src/hallview/data/db/*.db")
	if err != nil || len(dbFiles) == 0 {
		printText("No databases found. Please rebuild databases first.")
		printText("Press [Enter] to continue...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		clearScreen()
		return
	}

	// Gather all distinct room values across all dbs for fuzzy matching
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
			rows.Scan(&r)
			if r != "" {
				allRooms[r] = true
			}
		}
		rows.Close()
		db.Close()
	}

	reader := bufio.NewReader(os.Stdin)

	for {

		fmt.Print(indent + "Enter room name or keyword: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			clearScreen()
			return
		}

		clearScreen()
		printH1("Search by Room")
		fmt.Println()

		// Check for exact match first
		exactMatches := []string{}
		for r := range allRooms {
			if strings.EqualFold(r, input) {
				exactMatches = append(exactMatches, r)
			}
		}

		var targetRooms []string
		if len(exactMatches) > 0 {
			// direct
			targetRooms = exactMatches
		} else {
			// fuzzy match time
			type candidate struct {
				room  string
				score int
			}
			var candidates []candidate
			lowerInput := strings.ToLower(input)

			for r := range allRooms {
				lowerRoom := strings.ToLower(r)
				if strings.Contains(lowerRoom, lowerInput) {
					score := 100
					// STARTS WITH THE SAME THING = ++
					if strings.HasPrefix(lowerRoom, lowerInput) {
						score += 50
					}
					// SHORTER ROOM NAMES (more specific match) = +
					score += max(0, 20-len(r))
					candidates = append(candidates, candidate{room: r, score: score})
				}
			}

			// sort score down
			for i := 0; i < len(candidates); i++ {
				for j := i + 1; j < len(candidates); j++ {
					if candidates[j].score > candidates[i].score {
						candidates[i], candidates[j] = candidates[j], candidates[i]
					}
				}
			}

			if len(candidates) > 0 {
				printH2("Did you mean:", color.New(color.FgYellow))
				limit := min(12, len(candidates))
				for i := 0; i < limit; i++ {
					color.Cyan(fmt.Sprintf("   %s (score: %d)", candidates[i].room, candidates[i].score))
				}
				fmt.Println()
				printText("Showing results for closest match ^^.")
				fmt.Println()

				targetRooms = []string{candidates[0].room}
			} else {
				printTextf("No rooms found matching \"%s\".\n", input)
				printText("Press [Enter] to continue...")
				reader.ReadString('\n')
				clearScreen()
				continue
			}
		}

		dayNames := map[string]string{
			"2": "Monday",
			"3": "Tuesday",
			"4": "Wednesday",
			"5": "Thursday",
			"6": "Friday",
		}

		type entry struct {
			building string
			room     string
			start    int
			end      int
			day      string
			course   string
			section  string
		}
		seen := make(map[string]bool)
		results := []entry{}

		for _, dbFile := range dbFiles {
			db, err := sql.Open("sqlite", dbFile)
			if err != nil {
				errf("Failed to open %s: %v\n", dbFile, err)
				continue
			}

			for _, room := range targetRooms {
				rows, err := db.Query("SELECT building, room, start, end, day, course, section FROM schedule WHERE room = ? ORDER BY day, building, room, start, end, course", room)
				if err != nil {
					errf("Query failed for room=%s in %s: %v\n", room, dbFile, err)
					db.Close()
					continue
				}

				count := 0
				for rows.Next() {
					var e entry
					if err := rows.Scan(&e.building, &e.room, &e.start, &e.end, &e.day, &e.course, &e.section); err != nil {
						errf("Scan failed: %v\n", err)
						continue
					}
					key := fmt.Sprintf("%s|%s|%d|%d|%s|%s|%s", e.building, e.room, e.start, e.end, e.day, e.course, e.section)
					if !seen[key] {
						seen[key] = true
						results = append(results, e)
					}
					count++
				}
				if err := rows.Err(); err != nil {
					errf("Row iteration error: %v\n", err)
				}
				logf("%s room=%s -> %d rows\n", dbFile, room, count)
			}
			db.Close()
		}

		fmt.Println()
		if len(results) == 0 {
			printText("No schedules found for the selected room.")
		} else {
			printH2(fmt.Sprintf("Found %d result(s)", len(results)), color.New(color.FgGreen))
			fmt.Println()

			var lastDay string
			for _, e := range results {
				if e.day != lastDay {
					if lastDay != "" {
						printH3("")
					}
					lastDay = e.day
					dayLabel := dayNames[e.day]
					if dayLabel == "" {
						dayLabel = fmt.Sprintf("Day %s", e.day)
					}
					color.RGB(50,50,50).Println(fmt.Sprintf("\n  %s", dayLabel))
				}
				color.RGB(150, 150, 205).Printf("%s%s\n", indent, e.building)
				color.RGB(150, 150, 205).Printf("%sRoom: %s\n", indent, e.room)
				color.RGB(130, 205, 130).Printf("%sTime: %s - %s\n", indent, mm(e.start), mm(e.end))
				color.RGB(205, 170, 80).Printf("%sCourse: %s\n", indent, e.course)
				if e.section != "" {
					color.RGB(255, 180, 255).Printf("%sSection: %s\n", indent, e.section)
				}
				fmt.Println()
			}
		}

		// Compute free time blocks per day based on occupied intervals
		occupiedByDay := make(map[string][][2]int)
		for _, e := range results {
			occupiedByDay[e.day] = append(occupiedByDay[e.day], [2]int{e.start, e.end})
		}

		printH3("")
		printH2("Free Time Blocks", color.New(color.FgCyan))
		fmt.Println()

		dayOrder := []string{"2", "3", "4", "5", "6"}
		for _, day := range dayOrder {
			dayLabel := dayNames[day]
			intervals := occupiedByDay[day]

			// Sort occupied intervals by start time
			sort.Slice(intervals, func(i, j int) bool {
				return intervals[i][0] < intervals[j][0]
			})

			// Merge overlapping/adjacent occupied intervals
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

			// Compute free intervals between 00:00 (0) and 24:00 (1440)
			freeIntervals := [][2]int{}
			cursor := 0
			for _, iv := range merged {
				if iv[0] > cursor {
					freeIntervals = append(freeIntervals, [2]int{cursor, iv[0]})
				}
				if iv[1] > cursor {
					cursor = iv[1]
				}
			}
			if cursor < 1440 {
				freeIntervals = append(freeIntervals, [2]int{cursor, 1440})
			}

			// Filter out 10-minute blocks that span exactly :20 to :30
			filtered := [][2]int{}
			for _, iv := range freeIntervals {
				if iv[1]-iv[0] == 10 && iv[0]%60 == 20 && iv[1]%60 == 30 {
					continue
				}
				filtered = append(filtered, iv)
			}

			// Format in military time
			parts := []string{}
			for _, iv := range filtered {
				parts = append(parts, fmt.Sprintf("%s-%s", militaryTime(iv[0]), militaryTime(iv[1])))
			}

			if len(parts) == 0 {
				color.RGB(150, 150, 150).Printf("%s%s: (no free time)\n", indent, dayLabel)
			} else {
				color.RGB(130, 205, 130).Printf("%s%s: %s\n", indent, dayLabel, strings.Join(parts, ", "))
			}
		}

		fmt.Println()
		printText("Press [Enter] to return to menu, or type a new room to search:")
	}
}

func handle2() {}
func handle3() {}
func handle4() {}
func handle5() {}
func handle6() {}
func handle7() {}

func militaryTime(minutes int) string {
	minutes %= 1440
	if minutes < 0 {
		minutes += 1440
	}
	h := minutes / 60
	m := minutes % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

func mm(minutes int) string {
	minutes %= 1440
	h := minutes / 60
	m := minutes % 60

	period := "AM"
	if h >= 12 {
		period = "PM"
		if h > 12 {
			h -= 12
		}
	}
	if h == 0 {
		h = 12
	}

	return fmt.Sprintf("%d:%02d %s", h, m, period)
}