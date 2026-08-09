package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"path/filepath"
	"io/fs"

	"github.com/spf13/cobra"
	"github.com/fatih/color"
)

var TERM_LENGTH = 80

func initcheck() []bool {
	baseDirs := []string{"schedules", "courses", "state", "xml"}
	dataBaseDirs := map[string]bool{"schedules": true, "courses": true, "state": true}

	exists := make([]bool, len(baseDirs))
	for i, base := range baseDirs {
		exists[i] = true

		filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				rel, _ := filepath.Rel(base, path)
				if rel == "." || rel == "" {
					return nil
				}

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

func hasDatabases() bool {
	files, err := filepath.Glob("src/hallview/data/db/*.db")
	if err != nil {
		return false
	}
	return len(files) > 0
}

func printLoaded() error {
	root := "schedules"
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			printText(d.Name())
		}
		return nil
	})

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
				logf("%s folder OK...\n", name)
				if !ex[i] {
					logf("%s folder NOT OK...\n", name)
				}
			}
			fmt.Println()
			fmt.Println(`Press [Enter] to continue...`)

			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()

			clearScreen()
			for {
				printH1("home")
				fmt.Println()
				printH2("Detected Files", color.New(color.FgGreen))
				printLoaded()
				fmt.Println()
				printH2("Favorites", color.New(color.FgRed))
				printFavs()
				hasDB := hasDatabases()

				fmt.Println()
				printH2("What would you like to do?")
				if !hasDB {
					color.Yellow("No databases found. Please rebuild databases first.")
				}
				printText("0. Rebuild Databases")
				if hasDB {
					printText("1. Search by Room")
					printText("2. Search by Time")
					printText("3. Search by Course")
					printText("4. Custom SQL Query")
					printText("5. Manage Favorites")
				} else {
					color.RGB(211, 211, 211).Print(indent + "1. Search by Room\n")
					color.RGB(211, 211, 211).Print(indent + "2. Search by Time\n")
					color.RGB(211, 211, 211).Print(indent + "3. Search by Course\n")
					color.RGB(211, 211, 211).Print(indent + "4. Custom SQL Query\n")
					color.RGB(211, 211, 211).Print(indent + "5. Manage Favorites\n")
				}
				printText("6. Settings")
				printText("7. About")
				printText("8. Exit")
				fmt.Println()
				printH2("")
				printTextf("Enter your choice (1-7): ")

				scanner.Scan()
				choice := strings.TrimSpace(scanner.Text())

				if !hasDB && choice >= "1" && choice <= "5" {
					printText("No databases found. Please rebuild databases first.")
					printText("Press [Enter] to continue...")
					scanner.Scan()
					clearScreen()
					continue
				}

				switch choice {
				case "0":
					createSchedDB()
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
					handle7()
				case "8":
					fmt.Println("Exiting...")
					return
				default:
				printText("Invalid choice, please enter 0-8\n")
				printText("\n")

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
	os.Stdout.WriteString("\x1b[3;J\x1b[H\x1b[2J")
}