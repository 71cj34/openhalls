package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// stdinReader is shared so buffered input is never split across prompts.
var stdinReader = bufio.NewReader(os.Stdin)

// cooked ahh lines
// caller decides what empty input means!!! convention: empty = cancel/back
func readInputLine(prompt string) (string, bool) {
	fmt.Print(indent + prompt)
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", true
	}
	return strings.TrimSpace(line), false
}
