package main
import (
	"fmt"
	"bufio"
	"os"
)

func handle1() {
	clearScreen()
	fmt.Println(`=============================================`)
	fmt.Println(`                Option 1                    `)
	fmt.Println(`=============================================`)
	fmt.Println()
	fmt.Println(`This is the first option module.`)
	fmt.Println()
	fmt.Print("Press [Enter] to return to menu...")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	clearScreen()
}

func handle2() {
	clearScreen()
	fmt.Println(`=============================================`)
	fmt.Println(`                Option 2                    `)
	fmt.Println(`=============================================`)
	fmt.Println()
	fmt.Println(`This is the second option module.`)
	fmt.Println()
	fmt.Print("Press [Enter] to return to menu...")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	clearScreen()
}

func handle3() {
	clearScreen()
	fmt.Println(`=============================================`)
	fmt.Println(`                Option 3                    `)
	fmt.Println(`=============================================`)
	fmt.Println()
	fmt.Println(`This is the third option module.`)
	fmt.Println()
	fmt.Print("Press [Enter] to return to menu...")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	clearScreen()
}

func handle4() {}
func handle5() {}
func handle6() {}