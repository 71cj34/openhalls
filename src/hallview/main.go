package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/fatih/color"
)

var TERM_LENGTH = 80

func initcheck() []bool {
	baseDirs := []string{"schedules", "courses", "state", "xml"}
	dataBaseDirs := map[string]bool{"schedules": true, "courses": true, "state": true}

	exists := make([]bool, len(baseDirs))
	for i, base := range baseDirs {
		exists[i] = true // assume OK until proven otherwise

		filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				rel, _ := filepath.Rel(base, path)
				if rel == "." || rel == "" {
					return nil
				}
				// Check if this subfolder has the required files
				if dataBaseDirs[base] {
					files, _ := filepath.Glob(filepath.Join(path, "*.json"))
					if len(files) == 0 {
						exists[i] = false
					}
				} else {
					files, _ := filepath.Glob(filepath.Join(path, "*.xml"))
					if len(files) == 0 {
						exists[i] = false
					}
				}
			}
			return nil
		})
	}

	return exists
}

func main() {
	var logo = `
=======================================================================

    //   ) )                            //    / /
   //   / /  ___      ___       __     //___ / /  ___     // //  ___
  //   / / //   ) ) //___) ) //   ) ) / ___   / //   ) ) // // ((   ) )
 //   / / //___/ / //       //   / / //    / / //   / / // //   \ \
((___/ / //       ((____   //   / / //    / / ((___( ( // // //   ) )

=======================================================================
`
	var rootCmd = &cobra.Command{
		Use:   "hallview",
		Short: "Hallview CLI application",
		Long:  `A command-line interface for Hallview with multiple modules.`,
		Run: func(cmd *cobra.Command, args []string) {
			clearScreen()
			color.Yellow(logo)
			fmt.Println()
			color.HiRed("⚠  If you are running this program for the first time, please read the README.md file first! ⚠\n")

			ex := initcheck()
			colors := make([]*color.Color, len(ex))

			for i, val := range ex {
				if val {
					colors[i] = color.New(color.FgGreen)
				} else {
					colors[i] = color.New(color.FgRed)
				}
			}
			folders := []string{"Schedules", "Courses", "State", "XML"}
			for i, name := range folders {
				status := "OK"
				if !ex[i] {
					status = "NOT OK"
				}

				colors[i].Printf("[ • ]  ")
				fmt.Printf("%s folder %s...\n", name, status)
			}
			fmt.Println()
			fmt.Println(`Press [Enter] to continue...`)

			// Wait for user input
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()

			// Show main menu
			clearScreen()
			for {
				printH1("home")
				fmt.Println()
				printH2("Favorites", color.New(color.FgRed))
				printFavs()
				fmt.Println()
				printH2("What would you like to do?")
				printText("1. Search by Room")
				printText("2. Search by Time")
				printText("3. Search by Course")
				printText("4. Manage Favorites")
				printText("5. Settings")
				printText("6. About")
				printText("7. Exit")
				fmt.Println()
				printH2("")
				printTextf("Enter your choice (1-7): ")

				scanner.Scan()
				choice := strings.TrimSpace(scanner.Text())

				switch choice {
				case "1":
					handle1()
				case "2":
					handle2()
				case "3":
					handle3()
				case "4":
					handle4()
				case "5":
					handle5()
				case "6":
					handle6()
				case "7":
					fmt.Println("Exiting...")
					return
				default:
				printText("Invalid choice, please enter 1-4\n")
				printText("\n")
				// Pause before redrawing menu
				printText("Press [Enter] to continue...")
					scanner.Scan()
					clearScreen()
				}
			}
		},
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}

func clearScreen() {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "cls")
	default: // who is running ts on unix (wilted rose emoji)
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}