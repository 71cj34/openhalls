package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// stdinReader is shared so buffered input is never split across prompts.
var stdinReader = bufio.NewReader(os.Stdin)

// readInputLine prints prompt and reads one cooked line.
// The terminal handles arrows/edits; Enter submits.
// Returns back=true only on EOF/error. Empty input is "" with
// back=false — each caller decides whether empty means back
// or a default.
func readInputLine(prompt string) (string, bool) {
	fmt.Print(indent + prompt)
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", true
	}
	return strings.TrimSpace(line), false
}
