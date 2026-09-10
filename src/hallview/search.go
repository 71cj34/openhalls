package main

import (
	"bufio"
	"fmt"
	"sort"
	"strings"
)

// Shared fuzzy-pick: exact match wins, otherwise a numbered top-N
// shortlist. Room and course search both read through here so the
// "did you mean" UX stays identical.

type scoredPick struct {
	value string
	label string
	score int
}

func normKey(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

func scoreContains(value, input string) (bool, int) {
	nv, ni := normKey(value), normKey(input)
	if ni == "" || !strings.Contains(nv, ni) {
		return false, 0
	}
	score := 100
	if strings.HasPrefix(nv, ni) {
		score += 50
	}
	return true, score + max(0, 20-len(value))
}

// fuzzyPick resolves input against candidates. Labels carry extra
// context (course titles); matching is on values only. Returns the
// picked value, or "" when nothing matched.
func fuzzyPick(reader *bufio.Reader, noun string, candidates map[string]string, input string) (string, bool) {
	input = strings.TrimSpace(input)
	for v := range candidates {
		if strings.EqualFold(v, input) {
			return v, true
		}
	}

	var scored []scoredPick
	for v, label := range candidates {
		if ok, s := scoreContains(v, input); ok {
			scored = append(scored, scoredPick{value: v, label: label, score: s})
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].value < scored[j].value
	})

	if len(scored) == 0 {
		wrnf("No %ss matching %q.\n", noun, input)
		return "", false
	}

	const limit = 5
	showing := min(limit, len(scored))
	printH2(fmt.Sprintf("%d match(es), showing %d", len(scored), showing))
	rows := make([][]string, 0, showing)
	for i := 0; i < showing; i++ {
		rows = append(rows, []string{fmt.Sprintf("%d", i+1), scored[i].value, scored[i].label})
	}
	headers := []string{"#", "Course", "Title"}
	if noun == "room" {
		headers = []string{"#", "Room"}
		plain := make([][]string, 0, showing)
		for _, r := range rows {
			plain = append(plain, r[:2])
		}
		rows = plain
	}
	printTable(headers, rows)
	line, back := readInputLine(fmt.Sprintf("Pick 1-%d [1], or type to refine: ", showing))
	if back {
		return fuzzyPickBack(reader, noun, candidates)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return scored[0].value, true
	}
	var n int
	if _, err := fmt.Sscanf(line, "%d", &n); err == nil && n >= 1 && n <= showing {
		return scored[n-1].value, true
	}
	return fuzzyPick(reader, noun, candidates, line)
}

// fuzzyPickBack re-prompts the top-level input after a back-out from
// the pick list, so Backspace never drops straight to the home menu.
func fuzzyPickBack(reader *bufio.Reader, noun string, candidates map[string]string) (string, bool) {
	prompt := "Course code or keyword (Backspace = back): "
	if noun == "room" {
		prompt = "Room name or keyword (Backspace = back): "
	}
	fmt.Println()
	input, back := readInputLine(prompt)
	if back {
		return "", false
	}
	if strings.TrimSpace(input) == "" {
		return fuzzyPickBack(reader, noun, candidates)
	}
	return fuzzyPick(reader, noun, candidates, input)
}

func keysAsCandidates(keys map[string]bool) map[string]string {
	out := make(map[string]string, len(keys))
	for k := range keys {
		out[k] = ""
	}
	return out
}
