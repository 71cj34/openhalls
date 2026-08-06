package main

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

func printH1(txt string) {
	postfill := " " + strings.Repeat("=", TERM_LENGTH-1)
	color.RGB(50, 114, 255).Printf("| > /%s%s|\n", txt, strings.Repeat(" ", TERM_LENGTH-4-len(txt)))
	color.RGB(50, 114, 255).Println(postfill)
}


func printH2(txt string, col ...*color.Color) {

 	var c *color.Color
		if len(col) > 0 && col[0] != nil {
		    c = col[0]
		} else {
		    // color.RGB returns *color.Color
		    c = color.RGB(97, 214, 214)
		}


	prefill := " " + strings.Repeat("_", TERM_LENGTH - 1)
	c.Print(prefill + "\n")
	c.Printf("| %s%s|\n", txt, strings.Repeat(" ", TERM_LENGTH - 2 - len(txt)))
}

const indent = "   "

func printText(txt string) {
	fmt.Print(indent + txt + "\n")
}

func printTextf(format string, a ...interface{}) {
	fmt.Print(indent)
	fmt.Printf(format, a...)
}