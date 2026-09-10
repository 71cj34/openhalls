package main

import (
	"fmt"
	"sort"
	"strings"
)

func handle3() {
	printH1("Search by Course")

	if len(listDBFiles()) == 0 {
		wrnf("No databases found. Rebuild databases first (menu option 0).\n")
		pause()
		return
	}

	courses := collectCourses()
	if len(courses) == 0 {
		wrnf("Databases contain no courses.\n")
		pause()
		return
	}
	titles := courseTitles()
	candidates := make(map[string]string, len(courses))
	for c := range courses {
		candidates[c] = titles[c]
	}

	for {
		fmt.Println()
		input, back := readInputLine("Course code or keyword (empty = back): ")
		if back || strings.TrimSpace(input) == "" {
			return
		}

		course, ok := fuzzyPick("course", candidates, input)
		if !ok {
			continue
		}

		secFilter, back := readInputLine("Section filter [empty = all sections]: ")
		if back {
			return
		}

		runCourseSearch(course, secFilter, true)
	}
}

// runCourseSearch renders one saved or interactive course search without
// re-prompting. offerFav controls the trailing fav prompt.
func runCourseSearch(course, secFilter string, offerFav bool) {
	results, sources := queryCourse(course)
	if len(results) == 0 {
		wrnf("No classes found for %s.\n", course)
		return
	}
	filtered := results
	if strings.TrimSpace(secFilter) != "" {
		filtered = nil
		for _, e := range results {
			if strings.Contains(strings.ToLower(e.section), strings.ToLower(secFilter)) {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == 0 {
			wrnf("No sections of %s matching %q.\n", course, secFilter)
			return
		}
	}

	titles := courseTitles()
	printCourseReport(course, titles[course], filtered, sources)
	headers, rows := entryRows(filtered)
	offerExportRows("course", course, headers, rows)
	if offerFav {
		offerSaveFavorite(Favorite{Kind: "course", Course: course, SectionFilter: strings.TrimSpace(secFilter)})
	}
}

// printCourseReport groups a course's meetings by section, each with
// its weekly pattern on one line — the question is "when/where does
// C01 meet?", not "list every row".
func printCourseReport(course, title string, results []entry, sources []string) {
	head := course
	if title != "" {
		head += " — " + title
	}
	sections := groupSections(results)
	printH2(fmt.Sprintf("%s · %d section(s), %d meeting(s)", head, len(sections), len(results)))
	if len(sources) > 0 {
		dimStyle.Printf("%sTerms: %s\n", indent, strings.Join(sources, ", "))
	}

	for _, s := range sections {
		fmt.Println()
		sectionStyle.Printf("%s%s — %s\n", indent, s.section, s.pattern())
		rows := make([][]string, 0, len(s.rows))
		for _, e := range s.rows {
			rows = append(rows, []string{
				dayNames[e.day],
				fmt.Sprintf("%s–%s", fmtClock(e.start), fmtClock(e.end)),
				fmtDur(e.end - e.start),
				e.building,
				e.room,
			})
		}
		printTable([]string{"Day", "Time", "Dur", "Building", "Room"}, rows)
	}
	fmt.Println()
}

type sectionGroup struct {
	section string
	rows    []entry
}

// pattern compresses a section's meetings to "Mon 10:30 AM–11:20 AM ·
// BSB_147" when uniform, else "Mon/Wed/Fri · 2 rooms".
func (s sectionGroup) pattern() string {
	days := map[string]bool{}
	rooms := map[string]bool{}
	times := map[string]bool{}
	for _, e := range s.rows {
		if d, ok := dayNames[e.day]; ok {
			days[d] = true
		}
		rooms[e.room] = true
		times[fmt.Sprintf("%s–%s", fmtClock(e.start), fmtClock(e.end))] = true
	}
	dayList := make([]string, 0, len(days))
	for d := range days {
		dayList = append(dayList, d)
	}
	sort.Slice(dayList, func(i, j int) bool { return weekdayRank(dayList[i]) < weekdayRank(dayList[j]) })
	dayStr := strings.Join(dayList, "/")
	if dayStr == "Monday/Tuesday/Wednesday/Thursday/Friday" {
		dayStr = "Mon–Fri"
	} else {
		short := map[string]string{
			"Monday": "Mon", "Tuesday": "Tue", "Wednesday": "Wed",
			"Thursday": "Thu", "Friday": "Fri",
		}
		for i, d := range dayList {
			if s, ok := short[d]; ok {
				dayList[i] = s
			}
		}
		dayStr = strings.Join(dayList, "/")
	}
	switch {
	case len(times) == 1 && len(rooms) == 1:
		for t := range times {
			for r := range rooms {
				return fmt.Sprintf("%s %s · %s", dayStr, t, r)
			}
		}
	case len(times) == 1:
		for t := range times {
			return fmt.Sprintf("%s %s · %d rooms", dayStr, t, len(rooms))
		}
	default:
		return fmt.Sprintf("%s · %d meetings", dayStr, len(s.rows))
	}
	return dayStr
}

func weekdayRank(day string) int {
	switch day {
	case "Monday":
		return 0
	case "Tuesday":
		return 1
	case "Wednesday":
		return 2
	case "Thursday":
		return 3
	case "Friday":
		return 4
	}
	return 99
}

func groupSections(results []entry) []sectionGroup {
	bySection := make(map[string][]entry)
	for _, e := range results {
		bySection[e.section] = append(bySection[e.section], e)
	}
	out := make([]sectionGroup, 0, len(bySection))
	for sec, rows := range bySection {
		sort.Slice(rows, func(i, j int) bool {
			if ri, rj := dayRank(rows[i].day), dayRank(rows[j].day); ri != rj {
				return ri < rj
			}
			return rows[i].start < rows[j].start
		})
		out = append(out, sectionGroup{section: sec, rows: rows})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].section < out[j].section })
	return out
}
