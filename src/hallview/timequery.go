package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Two modes, picked up front so prompts stay short:
//   free — rooms with no class overlapping the window (default)
//   busy — rooms holding class during the window

type timeQuery struct {
	days     []string
	start    int
	end      int
	freeMode bool
	nameLike string
	limit    int
}

type roomHit struct {
	room     string
	building string
	until    int
	next     *entry
}

func handle2() {
	printH1("Search by Time")

	if len(listDBFiles()) == 0 {
		wrnf("No databases found. Rebuild databases first (menu option 0).\n")
		pause()
		return
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println()
		printText("Find rooms free (or busy) in a time window.")
		printText("Backspace on day/start = back to menu.")
		q, action := promptTimeQuery(reader)
		switch action {
		case queryBack:
			return
		case queryRetry:
			continue
		}

		results, sources := querySchedule("day IN ('" + strings.Join(q.days, "','") + "')")
		if len(results) == 0 {
			wrnf("No classes on record for those days.\n")
			continue
		}

		if q.freeMode {
			dayHits := printFreeReport(q, results, sources)
			headers, rows := freeHitsToRows(q.days, dayHits)
			label := "free-" + fmtClock(q.start) + "-" + fmtClock(q.end)
			if q.nameLike != "" {
				label += "-" + q.nameLike
			}
			offerExportRows("time", label, headers, rows)
		} else {
			dayRows := printBusyReport(q, results, sources)
			headers, rows := busyRowsToRows(q.days, dayRows)
			label := "busy-" + fmtClock(q.start) + "-" + fmtClock(q.end)
			if q.nameLike != "" {
				label += "-" + q.nameLike
			}
			offerExportRows("time", label, headers, rows)
		}
	}
}

func promptTimeQuery(reader *bufio.Reader) (timeQuery, queryAction) {
	q := timeQuery{limit: 30}

	line, ok := promptLine(reader, "Day(s) [Mon-Fri, e.g. Mon,Wed or Mon-Fri, Backspace = back]: ")
	if !ok {
		return q, queryBack
	}
	if strings.TrimSpace(line) == "" {
		wrnf("Enter a day (Backspace to go back).\n")
		return q, queryRetry
	}
	days, valid := parseDays(line, dayOrder)
	if !valid {
		wrnf("Could not parse days %q. Try Mon, Tue/Wed, Mon-Fri, or all.\n", line)
		return q, queryRetry
	}
	q.days = days

	line, ok = promptLine(reader, "Start [e.g. 10:30am, 13:30, Backspace = back]: ")
	if !ok {
		return q, queryBack
	}
	if strings.TrimSpace(line) == "" {
		wrnf("Enter a start time (Backspace to go back).\n")
		return q, queryRetry
	}
	start, err := parseClock(strings.TrimSpace(line))
	if err != nil {
		wrnf("Bad start time: %v\n", err)
		return q, queryRetry
	}
	q.start = start

	line, ok = promptLine(reader, fmt.Sprintf("End [%s, empty = +1h]: ", fmtClock(start+60)))
	if !ok {
		return q, queryBack
	}
	if strings.TrimSpace(line) == "" {
		q.end = start + 60
	} else {
		end, err := parseClock(strings.TrimSpace(line))
		if err != nil {
			wrnf("Bad end time: %v\n", err)
			return q, queryRetry
		}
		q.end = end
	}
	if q.end <= q.start {
		wrnf("End must be after start.\n")
		return q, queryRetry
	}

	line, ok = promptLine(reader, "Mode [(f)ree / (b)usy, default f]: ")
	if !ok {
		return q, queryBack
	}
	q.freeMode = true
	if s := strings.ToLower(strings.TrimSpace(line)); s == "b" || s == "busy" {
		q.freeMode = false
	}

	line, ok = promptLine(reader, "Room filter [empty = all rooms]: ")
	if !ok {
		return q, queryBack
	}
	q.nameLike = strings.TrimSpace(line)

	return q, queryOK
}

type queryAction int

const (
	queryOK queryAction = iota
	queryBack
	queryRetry
)

func promptLine(reader *bufio.Reader, label string) (string, bool) {
	_ = reader
	line, back := readInputLine(label)
	if back {
		return "", false
	}
	return line, true
}

// printFreeReport lists rooms free for the whole window, grouped by
// day, with "free until" so the reader can plan the next block.
// Returns per-day hits for CSV export (full list, not the 30-row view).
func printFreeReport(q timeQuery, results []entry, sources []string) map[string][]roomHit {
	index, buildingOf := roomDayIndex(results)
	title := "Free rooms"
	if q.nameLike != "" {
		title += fmt.Sprintf(" matching %q", q.nameLike)
	}
	printH2(fmt.Sprintf("%s · %s–%s", title, fmtClock(q.start), fmtClock(q.end)))
	if len(sources) > 0 {
		dimStyle.Printf("%sTerms: %s\n", indent, strings.Join(sources, ", "))
	}

	like := strings.ToLower(q.nameLike)
	dayHits := make(map[string][]roomHit, len(q.days))
	for _, day := range q.days {
		label := dayNames[day]
		var hits []roomHit
		for room, byDay := range index {
			if like != "" && !strings.Contains(strings.ToLower(room), like) {
				continue
			}
			if roomFreeOn(byDay[day], q.start, q.end) {
				hits = append(hits, roomHit{
					room:     room,
					building: buildingOf[room],
					until:    freeUntil(byDay[day], q.end),
					next:     nextAfter(byDay[day], q.end),
				})
			}
		}
		sort.Slice(hits, func(i, j int) bool {
			if hits[i].until != hits[j].until {
				return hits[i].until > hits[j].until
			}
			return hits[i].room < hits[j].room
		})

		fmt.Println()
		if len(hits) == 0 {
			warnStyle.Printf("%s%s — nothing free %s–%s\n", indent, label, fmtClock(q.start), fmtClock(q.end))
			continue
		}
		sectionStyle.Printf("%s%s — %d free\n", indent, label, len(hits))
		rows := make([][]string, 0, min(q.limit, len(hits)))
		for i, h := range hits {
			if i >= q.limit {
				break
			}
			next := "rest of day"
			if h.next != nil {
				next = fmt.Sprintf("%s (%s)", fmtClock(h.next.start), h.next.course)
			}
			rows = append(rows, []string{h.room, h.building, "until " + fmtClock(h.until), next})
		}
		printTable([]string{"Room", "Building", "Free", "Next class"}, rows)
		if len(hits) > q.limit {
			dimStyle.Printf("%s… and %d more (refine with a room filter)\n", indent, len(hits)-q.limit)
		}
		if len(hits) > 0 {
			dayHits[day] = hits
		}
	}
	fmt.Println()
	return dayHits
}

// printBusyReport lists classes overlapping the window, grouped by
// day — the inverse question ("what's on right now / sit in on?").
// Returns per-day display rows for CSV export (full list).
func printBusyReport(q timeQuery, results []entry, sources []string) map[string][][]string {
	title := "Occupied rooms"
	if q.nameLike != "" {
		title += fmt.Sprintf(" matching %q", q.nameLike)
	}
	printH2(fmt.Sprintf("%s · %s–%s", title, fmtClock(q.start), fmtClock(q.end)))
	if len(sources) > 0 {
		dimStyle.Printf("%sTerms: %s\n", indent, strings.Join(sources, ", "))
	}

	like := strings.ToLower(q.nameLike)
	dayRows := make(map[string][][]string, len(q.days))
	for _, day := range q.days {
		label := dayNames[day]
		var rows [][]string
		for _, e := range results {
			if e.day != day {
				continue
			}
			if like != "" && !strings.Contains(strings.ToLower(e.room), like) {
				continue
			}
			if !overlaps(e.start, e.end, q.start, q.end) {
				continue
			}
			rows = append(rows, []string{
				fmt.Sprintf("%s–%s", fmtClock(e.start), fmtClock(e.end)),
				e.room,
				e.course,
				e.section,
			})
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i][0] != rows[j][0] {
				return rows[i][0] < rows[j][0]
			}
			return rows[i][1] < rows[j][1]
		})

		fmt.Println()
		if len(rows) == 0 {
			okStyle.Printf("%s%s — nothing on %s–%s\n", indent, label, fmtClock(q.start), fmtClock(q.end))
			continue
		}
		sectionStyle.Printf("%s%s — %d class(es)\n", indent, label, len(rows))
		if len(rows) > 0 {
			full := make([][]string, len(rows))
			copy(full, rows)
			dayRows[day] = full
		}
		if len(rows) > q.limit {
			dimStyle.Printf("%sShowing %d of %d (refine with a room filter)\n", indent, q.limit, len(rows))
			rows = rows[:q.limit]
		}
		printTable([]string{"Time", "Room", "Course", "Section"}, rows)
	}
	fmt.Println()
	return dayRows
}

var dayAliases = map[string]string{
	"m": "2", "mon": "2", "monday": "2",
	"tu": "3", "tue": "3", "tues": "3", "tuesday": "3",
	"w": "4", "wed": "4", "weds": "4", "wednesday": "4",
	"th": "5", "thu": "5", "thur": "5", "thurs": "5", "thursday": "5",
	"f": "6", "fri": "6", "friday": "6",
}

// parseDays accepts "Mon", "Mon,Wed", "Mon-Fri", "weekdays", "all".
// It returns (nil, true) when input is empty, meaning "caller should
// retry" — distinct from (days, false) which is a hard parse error.
func parseDays(input string, fallback []string) ([]string, bool) {
	s := strings.ToLower(strings.TrimSpace(input))
	if s == "" {
		return append([]string(nil), fallback...), true
	}
	if s == "all" || s == "weekdays" || s == "mon-fri" || s == "m-f" {
		return append([]string(nil), fallback...), true
	}

	seen := make(map[string]bool)
	var out []string
	add := func(d string) {
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			from, ok1 := dayAliases[strings.TrimSpace(bounds[0])]
			to, ok2 := dayAliases[strings.TrimSpace(bounds[1])]
			if !ok1 || !ok2 {
				return nil, false
			}
			fi, ti := dayRank(from), dayRank(to)
			if fi > ti {
				return nil, false
			}
			for _, d := range fallback {
				if r := dayRank(d); r >= fi && r <= ti {
					add(d)
				}
			}
			continue
		}
		if d, ok := dayAliases[part]; ok {
			add(d)
			continue
		}
		return nil, false
	}
	if len(out) == 0 {
		return nil, false
	}
	sort.Slice(out, func(i, j int) bool { return dayRank(out[i]) < dayRank(out[j]) })
	return out, true
}

// parseClock accepts "10:30am", "10:30 am", "13:30", "9", "9pm".
func parseClock(input string) (int, error) {
	s := strings.ToLower(strings.TrimSpace(input))
	s = strings.ReplaceAll(s, " ", "")
	if s == "" {
		return 0, fmt.Errorf("empty time")
	}

	ampm := ""
	if strings.HasSuffix(s, "am") || strings.HasSuffix(s, "pm") {
		ampm = s[len(s)-2:]
		s = s[:len(s)-2]
	}

	hour, minute := 0, 0
	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		h, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("bad hour in %q", input)
		}
		m, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("bad minute in %q", input)
		}
		hour, minute = h, m
	} else {
		h, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("bad time %q", input)
		}
		hour = h
	}

	if minute < 0 || minute > 59 {
		return 0, fmt.Errorf("minute out of range in %q", input)
	}
	switch ampm {
	case "am":
		if hour < 1 || hour > 12 {
			return 0, fmt.Errorf("hour out of range in %q", input)
		}
		if hour == 12 {
			hour = 0
		}
	case "pm":
		if hour < 1 || hour > 12 {
			return 0, fmt.Errorf("hour out of range in %q", input)
		}
		if hour != 12 {
			hour += 12
		}
	default:
		if hour < 0 || hour > 23 {
			return 0, fmt.Errorf("hour out of range in %q", input)
		}
	}

	return hour*60 + minute, nil
}
