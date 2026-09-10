package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Favorite is one saved search. Kind is one of room, time, course, sql.
type Favorite struct {
	Name          string   `json:"name"`
	Kind          string   `json:"kind"`
	Room          string   `json:"room,omitempty"`
	Course        string   `json:"course,omitempty"`
	SectionFilter string   `json:"section_filter,omitempty"`
	Days          []string `json:"days,omitempty"`
	Start         int      `json:"start,omitempty"`
	End           int      `json:"end,omitempty"`
	FreeMode      bool     `json:"free_mode,omitempty"`
	NameLike      string   `json:"name_like,omitempty"`
	SQL           string   `json:"sql,omitempty"`
	Targets       []string `json:"targets,omitempty"`
}

func favsPath() string {
	return filepath.Join(repoRoot(), "src", "hallview", "data", "favs.json")
}

func loadFavorites() []Favorite {
	data, err := os.ReadFile(favsPath())
	if err != nil {
		return nil
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}
	var favs []Favorite
	if err := json.Unmarshal(data, &favs); err != nil {
		return nil
	}
	return favs
}

func saveFavorites(favs []Favorite) error {
	if favs == nil {
		favs = []Favorite{}
	}
	if err := os.MkdirAll(filepath.Dir(favsPath()), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(favs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(favsPath(), append(data, '\n'), 0644)
}

// favHotkey maps index 0-25 to a-z.
func favHotkey(i int) string {
	if i < 0 || i >= 26 {
		return ""
	}
	return string(rune('a' + i))
}

func favHotkeyIndex(s string) (int, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) != 1 {
		return 0, false
	}
	c := s[0]
	if c < 'a' || c > 'z' {
		return 0, false
	}
	return int(c - 'a'), true
}

func favDescribe(f Favorite) string {
	switch f.Kind {
	case "room":
		return fmt.Sprintf("[room] %s", f.Room)
	case "course":
		if f.SectionFilter != "" {
			return fmt.Sprintf("[course] %s (%s)", f.Course, f.SectionFilter)
		}
		return fmt.Sprintf("[course] %s", f.Course)
	case "time":
		days := favDayLabels(f.Days)
		mode := "free"
		if !f.FreeMode {
			mode = "busy"
		}
		filter := ""
		if f.NameLike != "" {
			filter = fmt.Sprintf(", %q", f.NameLike)
		}
		return fmt.Sprintf("[time] %s %s-%s %s%s", days, fmtClock(f.Start), fmtClock(f.End), mode, filter)
	case "sql":
		q := strings.Join(strings.Fields(f.SQL), " ")
		if len(q) > 60 {
			q = q[:60] + "..."
		}
		return fmt.Sprintf("[sql] %s", q)
	default:
		return fmt.Sprintf("[%s]", f.Kind)
	}
}

func favDayLabels(days []string) string {
	var labels []string
	short := map[string]string{"2": "Mon", "3": "Tue", "4": "Wed", "5": "Thu", "6": "Fri"}
	for _, d := range days {
		if s, ok := short[d]; ok {
			labels = append(labels, s)
		} else if n, ok := dayNames[d]; ok {
			labels = append(labels, n)
		} else {
			labels = append(labels, d)
		}
	}
	if len(labels) == 0 {
		return "?"
	}
	return strings.Join(labels, ",")
}

func favDefaultName(f Favorite) string {
	switch f.Kind {
	case "room":
		return "Room " + f.Room
	case "course":
		if f.SectionFilter != "" {
			return fmt.Sprintf("%s %s", f.Course, f.SectionFilter)
		}
		return f.Course
	case "time":
		mode := "free"
		if !f.FreeMode {
			mode = "busy"
		}
		return fmt.Sprintf("%s %s-%s %s", favDayLabels(f.Days), fmtClock(f.Start), fmtClock(f.End), mode)
	case "sql":
		q := strings.Join(strings.Fields(f.SQL), " ")
		if len(q) > 40 {
			q = q[:40] + "..."
		}
		return "SQL " + q
	}
	return string(f.Kind)
}

func printFavs() {
	favs := loadFavorites()
	if len(favs) == 0 {
		printText("No favorites yet")
		printText(`Type "fav" after a search to save it here.`)
		return
	}
	for i, f := range favs {
		name := f.Name
		if name == "" {
			name = favDefaultName(f)
		}
		key := favHotkey(i)
		if key == "" {
			printText(fmt.Sprintf("- %s — %s", name, favDescribe(f)))
			continue
		}
		printText(fmt.Sprintf("%s. %s — %s", key, name, favDescribe(f)))
	}
}

// offerSaveFavorite prompts after a completed search. Type fav to save.
func offerSaveFavorite(proto Favorite) {
	fmt.Println()
	line, back := readInputLine(`Type "fav" to save as favorite [Enter = continue]: `)
	if back {
		return
	}
	if strings.ToLower(strings.TrimSpace(line)) != "fav" {
		return
	}
	def := favDefaultName(proto)
	name, back := readInputLine(fmt.Sprintf("Favorite name [%s]: ", def))
	if back {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = def
	}
	proto.Name = name
	favs := loadFavorites()
	favs = append(favs, proto)
	if err := saveFavorites(favs); err != nil {
		errf("Could not save favorite: %v\n", err)
		return
	}
	okStyle.Printf("%sSaved favorite %q.", indent, name)
	if hk := favHotkey(len(favs) - 1); hk != "" {
		fmt.Printf(" Press %s on the home screen to re-run it.\n", hk)
	} else {
		fmt.Printf(" Use Manage Favorites to re-run it.\n")
	}
}

// runFavorite jumps to the equivalent search screen with saved params.
func runFavorite(fav Favorite) {
	switch fav.Kind {
	case "room":
		printH1("Search by Room")
		runRoomSearch(fav.Room, false)
		pause()
	case "course":
		printH1("Search by Course")
		runCourseSearch(fav.Course, fav.SectionFilter, false)
		pause()
	case "time":
		printH1("Search by Time")
		runTimeQuery(timeQuery{
			days:     append([]string(nil), fav.Days...),
			start:    fav.Start,
			end:      fav.End,
			freeMode: fav.FreeMode,
			nameLike: fav.NameLike,
			limit:    30,
		})
	case "sql":
		printH1("Custom SQL Query")
		runSQLSearch(fav.SQL, resolveFavTargets(fav.Targets))
	default:
		wrnf("Unknown favorite kind %q.\n", fav.Kind)
		pause()
	}
}

func resolveFavTargets(bases []string) []string {
	all := listDBFiles()
	if len(bases) == 0 {
		return all
	}
	want := make(map[string]bool, len(bases))
	for _, b := range bases {
		want[strings.ToLower(filepath.Base(b))] = true
		want[strings.ToLower(b)] = true
	}
	var out []string
	for _, f := range all {
		if want[strings.ToLower(filepath.Base(f))] || want[strings.ToLower(f)] {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		return all
	}
	return out
}

func favTargetBases(targets []string) []string {
	bases := make([]string, 0, len(targets))
	for _, t := range targets {
		bases = append(bases, filepath.Base(t))
	}
	return bases
}

// manageFavorites lists, renames, and deletes favorites.
func manageFavorites() {
	printH1("Manage Favorites")
	for {
		favs := loadFavorites()
		if len(favs) == 0 {
			wrnf("No favorites yet. Run a search, then type fav to save one.\n")
			pause()
			return
		}
		fmt.Println()
		for i, f := range favs {
			name := f.Name
			if name == "" {
				name = favDefaultName(f)
			}
			printText(fmt.Sprintf("%d. %s — %s", i+1, name, favDescribe(f)))
		}
		fmt.Println()
		printText("Commands: d <n> delete, r <n> rename, empty = back.")
		line, back := readInputLine("Manage favorites: ")
		if back || strings.TrimSpace(line) == "" {
			return
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			wrnf("Usage: d <n> or r <n> [new name].\n")
			continue
		}
		var n int
		if _, err := fmt.Sscanf(fields[1], "%d", &n); err != nil || n < 1 || n > len(favs) {
			wrnf("Pick a number 1-%d.\n", len(favs))
			continue
		}
		idx := n - 1
		switch strings.ToLower(fields[0]) {
		case "d", "del", "delete", "rm":
			removed := favs[idx].Name
			favs = append(favs[:idx], favs[idx+1:]...)
			if err := saveFavorites(favs); err != nil {
				errf("Could not save favorites: %v\n", err)
				continue
			}
			okStyle.Printf("%sRemoved %q.\n", indent, removed)
		case "r", "rename", "edit":
			cur := favs[idx].Name
			if cur == "" {
				cur = favDefaultName(favs[idx])
			}
			nn := ""
			if len(fields) > 2 {
				rest := strings.TrimSpace(line[len(fields[0]):])
				rest = strings.TrimSpace(strings.TrimPrefix(rest, fields[1]))
				nn = strings.TrimSpace(rest)
			} else {
				var back bool
				nn, back = readInputLine(fmt.Sprintf("New name [%s]: ", cur))
				if back {
					continue
				}
			}
			nn = strings.TrimSpace(nn)
			if nn == "" {
				nn = cur
			}
			favs[idx].Name = nn
			if err := saveFavorites(favs); err != nil {
				errf("Could not save favorites: %v\n", err)
				continue
			}
			okStyle.Printf("%sRenamed to %q.\n", indent, nn)
		default:
			wrnf("Unknown command %q. Use d <n> or r <n>.\n", fields[0])
		}
	}
}
