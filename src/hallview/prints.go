package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

const indent = "   "

var (
	verbose = os.Getenv("HALLVIEW_VERBOSE") == "1"

	titleStyle   = color.New(color.Bold, color.FgWhite)
	sectionStyle = color.New(color.Bold, color.FgCyan)
	dimStyle     = color.New(color.Faint)
	okStyle      = color.New(color.FgGreen)
	failStyle    = color.New(color.FgRed)
	warnStyle    = color.New(color.FgYellow)
)

// termWidth returns the usable output width, clamped to a sane range.
// Fixed 80-col output is what breaks on narrow terminals and wastes
// wide ones; COLUMNS lets users/pagers override.
func termWidth() int {
	if raw := strings.TrimSpace(os.Getenv("COLUMNS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 40 {
			if n > 120 {
				return 120
			}
			return n
		}
	}
	return 100
}

func rule(ch string) {
	w := termWidth()
	fmt.Println(dimStyle.Sprint(strings.Repeat(ch, w)))
}

// printH1 is the screen title / breadcrumb. Single style everywhere.
func printH1(txt string) {
	rule("─")
	titleStyle.Printf("› %s\n", txt)
	rule("─")
}

// printH2 is a section header. The optional color arg is kept for
// backwards compatibility but ignored: one style beats five.
func printH2(txt string, _ ...*color.Color) {
	if txt == "" {
		rule("─")
		return
	}
	fmt.Println()
	sectionStyle.Printf("── %s\n", txt)
}

// printH3 is a faint divider, optionally with dim trailing text.
func printH3(txt string, _ ...*color.Color) {
	if txt == "" {
		dimStyle.Println(strings.Repeat("·", termWidth()/2))
		return
	}
	dimStyle.Printf("%s%s\n", indent, txt)
}

func printText(txt string) {
	fmt.Print(indent + txt + "\n")
}

func printTextf(format string, a ...interface{}) {
	fmt.Print(indent)
	fmt.Printf(format, a...)
}

// printStatus renders one initcheck line: "✓ Schedules" instead of
// the old OK... / NOT OK... double-log noise.
func printStatus(ok bool, label string) {
	mark := "✓"
	style := okStyle
	if !ok {
		mark = "✗"
		style = failStyle
	}
	fmt.Printf("%s%s %s\n", indent, style.Sprint(mark), label)
}

// printTable renders rows as one-line-per-entry with an aligned
// header. This replaces the old 5-line rainbow card per class,
// which is what made large result sets unreadable.
func printTable(headers []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, r := range rows {
		for i := range headers {
			if i < len(r) && len(r[i]) > widths[i] {
				widths[i] = len(r[i])
			}
		}
	}
	// Shrink the widest column if the table overflows the terminal.
	for {
		total := len(indent)
		for _, w := range widths {
			total += w + 3
		}
		if total <= termWidth()+3 || len(widths) == 0 {
			break
		}
		big := 0
		for i := range widths {
			if widths[i] > widths[big] {
				big = i
			}
		}
		if widths[big] <= 10 {
			break
		}
		widths[big]--
	}

	pad := func(s string, w int) string {
		if len(s) > w {
			if w > 3 {
				return s[:w-1] + "…"
			}
			return s[:w]
		}
		return s + strings.Repeat(" ", w-len(s))
	}

	var b strings.Builder
	b.WriteString(indent)
	for i, h := range headers {
		if i > 0 {
			b.WriteString("   ")
		}
		b.WriteString(pad(strings.ToUpper(h), widths[i]))
	}
	titleStyle.Println(b.String())

	b.Reset()
	b.WriteString(indent)
	for i := range headers {
		if i > 0 {
			b.WriteString("   ")
		}
		b.WriteString(strings.Repeat("-", widths[i]))
	}
	dimStyle.Println(b.String())

	for _, r := range rows {
		b.Reset()
		b.WriteString(indent)
		for i := range headers {
			if i > 0 {
				b.WriteString("   ")
			}
			cell := ""
			if i < len(r) {
				cell = r[i]
			}
			b.WriteString(pad(cell, widths[i]))
		}
		fmt.Println(b.String())
	}
}

// fmtClock renders minutes-since-midnight in one consistent 12h style.
func fmtClock(minutes int) string {
	minutes %= 1440
	if minutes < 0 {
		minutes += 1440
	}
	h, m := minutes/60, minutes%60
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

// fmtDur renders a minute count compactly: 50m, 1h, 1h20m.
func fmtDur(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%dh", minutes/60)
	}
	return fmt.Sprintf("%dh%02dm", minutes/60, minutes%60)
}

func logf(format string, a ...interface{}) {
	if !verbose {
		return
	}
	dimStyle.Print("[LOG] ")
	fmt.Printf(format, a...)
}

func wrnf(format string, a ...interface{}) {
	warnStyle.Print("[WRN] ")
	fmt.Printf(format, a...)
}

func errf(format string, a ...interface{}) {
	failStyle.Print("[ERR] ")
	fmt.Printf(format, a...)
}
