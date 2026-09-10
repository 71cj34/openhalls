package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

const customRowLimit = 100

func runCustomQuery() {
	printH1("Custom SQL Query")

	dbFiles := listDBFiles()
	if len(dbFiles) == 0 {
		wrnf("No databases found. Rebuild databases first (menu option 0).\n")
		pause()
		return
	}

	printCustomHelp(dbFiles)

	targets := pickCustomTargets(dbFiles)
	if targets == nil {
		return
	}

	for {
		fmt.Println()
		line, back := readInputLine("SQL [empty = back, help = info]: ")
		if back || strings.TrimSpace(line) == "" {
			return
		}
		switch strings.ToLower(line) {
		case "help", "schema", "info", "example", "examples":
			printCustomHelp(dbFiles)
			continue
		}

		query, ok := normalizeCustomSQL(line)
		if !ok {
			wrnf("Only a single SELECT (or WITH...SELECT) statement is allowed.\n")
			continue
		}

		runSQLReport(targets, query, true)
	}
}

// runSQLReport runs one SQL string, prints, offers export, and
// optionally offers to favorite it. Shared by interactive + favorites.
func runSQLReport(targets []string, query string, offerFav bool) {
	cols, rows, full, sources, deduped := execCustomAcrossDBs(targets, query)
	if cols == nil {
		return
	}
	if len(rows) == 0 {
		wrnf("Query ran on %d database(s), 0 rows returned.\n", len(targets))
		return
	}
	printH2(fmt.Sprintf("Result · %d row(s)%s", len(full), customTruncNote()))
	if n := queryLimit(query); n > 0 && deduped > 0 && len(full) < n {
		wrnf("Query has LIMIT %d but only %d unique row(s) shown; %d duplicate row(s) merged.\n", n, len(full), deduped)
		printText("Identical rows are merged. Use SELECT DISTINCT or add columns (e.g. start, end, section) to keep rows distinct.")
	}
	if len(sources) > 0 {
		dimStyle.Printf("%sTerms: %s\n", indent, strings.Join(sources, ", "))
	}
	printTable(cols, rows)
	offerExportRows("sql", "query", cols, full)
	if offerFav {
		offerSaveFavorite(Favorite{Kind: "sql", SQL: query, Targets: favTargetBases(targets)})
	}
}

// runSQLSearch executes a saved SQL favorite without re-prompting or
// offering to re-save.
func runSQLSearch(query string, targets []string) {
	if targets == nil {
		targets = listDBFiles()
	}
	if len(targets) == 0 {
		wrnf("No databases found. Rebuild databases first (menu option 0).\n")
		pause()
		return
	}
	runSQLReport(targets, query, false)
	pause()
}

var customTruncated bool

func customTruncNote() string {
	if customTruncated {
		return fmt.Sprintf(" (showing first %d)", customRowLimit)
	}
	return ""
}

func printCustomHelp(dbFiles []string) {
	printH2("CAUTION: Advanced!")
	printText("This option runs raw SQL against the schedule databases.")
	printText("Bad queries return an error. No data is changed.")
	fmt.Println()
	printText("Table: schedule")
	printText("Columns: building TEXT, room TEXT, start INTEGER,")
	printText("         end INTEGER, day TEXT, course TEXT, section TEXT")
	fmt.Println()
	printText("  		start/end: minutes since midnight (630 = 10:30 AM).")
	printText("  		day: 2=Monday, 3=Tuesday, 4=Wednesday, 5=Thursday, 6=Friday.")
	printText("  		course: e.g. ABLD-3CD3.")
	printText("			section: e.g. C01 or L01.")
	fmt.Println()
	printText("Rules: single SELECT (or WITH...SELECT) only, one statement,")
	printText("no semicolon stacking. Capped at 100 rows.")
	fmt.Println()
	names := make([]string, 0, len(dbFiles))
	for _, f := range dbFiles {
		names = append(names, filepath.Base(f))
	}
	printText("Available Databases (one per term):")
	printText(strings.Join(names, ", "))
	fmt.Println()
	printText("Examples:")
	printText("  SELECT building, room, course FROM schedule WHERE day='2' LIMIT 10")
	printText("  SELECT room, start, end, course FROM schedule WHERE room LIKE '%147%' AND day='4' ORDER BY start")
	printText("  SELECT DISTINCT building FROM schedule")
	printText("  SELECT course, COUNT(*) AS n FROM schedule GROUP BY course ORDER BY n DESC LIMIT 10")
}

func pickCustomTargets(dbFiles []string) []string {
	if len(dbFiles) == 1 {
		return dbFiles
	}
	fmt.Println()
	printText("Query which database [empty = all terms, 0 = menu]:")
	for i, f := range dbFiles {
		printText(fmt.Sprintf("  %d. %s", i+1, filepath.Base(f)))
	}
	line, back := readInputLine("Choice: ")
	if back || strings.TrimSpace(line) == "0" {
		return nil
	}
	if strings.TrimSpace(line) == "" {
		return dbFiles
	}
	if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(dbFiles) {
		return []string{dbFiles[n-1]}
	}
	for _, f := range dbFiles {
		if strings.EqualFold(filepath.Base(f), line) || strings.EqualFold(f, line) {
			return []string{f}
		}
	}
	wrnf("Unknown database %q, querying all terms.\n", line)
	return dbFiles
}

func normalizeCustomSQL(input string) (string, bool) {
	q := strings.TrimSpace(input)
	q = strings.TrimRight(q, " \t\r\n;")
	if strings.TrimSpace(q) == "" || strings.Contains(q, ";") {
		return "", false
	}
	up := strings.ToUpper(q)
	if strings.HasPrefix(up, "SELECT") || strings.HasPrefix(up, "WITH") {
		return q, true
	}
	return "", false
}

var limitRe = regexp.MustCompile(`(?i)\bLIMIT\s+(\d+)`)

// queryLimit returns the trailing LIMIT n of a single SELECT, or -1.
// execCustomAcrossDBs runs the query once per term DB and merges the
// rows, so LIMIT applies per DB, not to the merged result.
func queryLimit(query string) int {
	m := limitRe.FindAllStringSubmatch(query, -1)
	if len(m) == 0 {
		return -1
	}
	n, err := strconv.Atoi(m[len(m)-1][1])
	if err != nil {
		return -1
	}
	return n
}

func execCustomAcrossDBs(dbFiles []string, query string) ([]string, [][]string, [][]string, []string, int) {
	var cols []string
	var out [][]string
	var full [][]string
	var sources []string
	seen := make(map[string]bool)
	customTruncated = false
	deduped := 0

	for _, dbFile := range dbFiles {
		db, err := sql.Open("sqlite", dbFile)
		if err != nil {
			errf("Failed to open %s: %v\n", dbFile, err)
			continue
		}
		rows, err := db.Query(query)
		if err != nil {
			errf("Query failed in %s: %v\n", filepath.Base(dbFile), err)
			db.Close()
			if cols == nil {
				return nil, nil, nil, nil, 0
			}
			continue
		}
		if cols == nil {
			cols, err = rows.Columns()
			if err != nil {
				errf("Could not read columns: %v\n", err)
				rows.Close()
				db.Close()
				return nil, nil, nil, nil, 0
			}
		}
		hit := false
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				continue
			}
			cells := make([]string, len(cols))
			raw := make([]string, len(cols))
			for i := range vals {
				cells[i] = customCell(cols[i], vals[i])
				raw[i] = rawCell(vals[i])
			}
			key := strings.Join(cells, "\x1f")
			if seen[key] {
				deduped++
				hit = true
				continue
			}
			seen[key] = true
			full = append(full, raw)
			if len(out) < customRowLimit {
				out = append(out, cells)
			} else {
				customTruncated = true
			}
			hit = true
		}
		rows.Close()
		db.Close()
		if hit {
			sources = append(sources, filepath.Base(dbFile))
		}
	}
	if cols == nil {
		return nil, nil, nil, nil, 0
	}
	return cols, out, full, sources, deduped
}

func rawCell(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func customCell(col string, v any) string {
	switch t := v.(type) {
	case nil:
		return "NULL"
	case string:
		return customNamed(col, t)
	case []byte:
		return customNamed(col, string(t))
	case int64:
		return customInt(col, t)
	case int:
		return customInt(col, int64(t))
	case float64:
		if t == float64(int64(t)) {
			return customInt(col, int64(t))
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func customInt(col string, v int64) string {
	switch strings.ToLower(col) {
	case "start", "end":
		return fmt.Sprintf("%s (%d)", fmtClock(int(v)), v)
	case "day":
		if label, ok := dayNames[strconv.FormatInt(v, 10)]; ok {
			return fmt.Sprintf("%s (%d)", label, v)
		}
	}
	return strconv.FormatInt(v, 10)
}

func customNamed(col, v string) string {
	if strings.EqualFold(col, "day") {
		if label, ok := dayNames[strings.TrimSpace(v)]; ok {
			return fmt.Sprintf("%s (%s)", label, v)
		}
	}
	if (strings.EqualFold(col, "start") || strings.EqualFold(col, "end")) && v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return fmt.Sprintf("%s (%s)", fmtClock(n), v)
		}
	}
	return v
}
