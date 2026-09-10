package main

import (
	"fmt"
	"sort"
	"strings"
)

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

// DRY fuzzy match thing
func fuzzyPick(noun string, candidates map[string]string, input string) (string, bool) {
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
	line, back := readInputLine(fmt.Sprintf("Pick 1-%d [0 = back or type to search again]: ", showing))
	if back || strings.TrimSpace(line) == "0" {
		return fuzzyPickBack(noun, candidates)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return scored[0].value, true
	}
	var n int
	if _, err := fmt.Sscanf(line, "%d", &n); err == nil && n >= 1 && n <= showing {
		return scored[n-1].value, true
	}
	return fuzzyPick(noun, candidates, line)
}

// backs up after the user goes back
func fuzzyPickBack(noun string, candidates map[string]string) (string, bool) {
	prompt := "Course code or keyword (empty = back): "
	if noun == "room" {
		prompt = "Room name or keyword (empty = back): "
	}
	fmt.Println()
	input, back := readInputLine(prompt)
	if back || strings.TrimSpace(input) == "" {
		return "", false
	}
	return fuzzyPick(noun, candidates, input)
}

func keysAsCandidates(keys map[string]bool) map[string]string {
	out := make(map[string]string, len(keys))
	for k := range keys {
		out[k] = ""
	}
	return out
}
