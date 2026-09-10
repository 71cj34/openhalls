package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"path/filepath"
	"io/fs"
	"time"

	"github.com/spf13/cobra"
	"github.com/fatih/color"
)

func initcheck() []bool {
	type dirCheck struct {
		dir string
		ext string
	}
	checks := []dirCheck{
		{scheduleDir(), "*.json"},
		{coursesDir(), "*.json"},
		{stateDir(), "*.json"},
		{xmlDir(), "*.xml"},
	}

	exists := make([]bool, len(checks))
	for i, c := range checks {
		exists[i] = true

		if st, err := os.Stat(c.dir); err != nil || !st.IsDir() {
			exists[i] = false
			continue
		}

		filepath.Walk(c.dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				rel, _ := filepath.Rel(c.dir, path)
				if rel == "." || rel == "" {
					return nil
				}

				files, _ := filepath.Glob(filepath.Join(path, c.ext))
				if len(files) == 0 {
					exists[i] = false
				}
			}
			return nil
		})
	}

	return exists
}

func hasDatabases() bool {
	return len(listDBFiles()) > 0
}

func printLoaded() error {
	root := scheduleDir()
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
     //   ) )                            //    / /
    //   / /  ___      ___       __     //___ / /  ___     // //  ___
   //   / / //   ) ) //___) ) //   ) ) / ___   / //   ) ) // // ((   ) )
  //   / / //___/ / //       //   / / //    / / //   / / // //   \ \
 ((___/ / //       ((____   //   / / //    / / ((___( ( // // //   ) )
`

	var rootCmd = &cobra.Command{
		Use:   "hallview",
		Short: "Hallview CLI application",
		Long:  `Find free rooms and lecture times from MyTimetable data.`,
		Run: func(cmd *cobra.Command, args []string) {
			color.Yellow("%s", logo)
			fmt.Println()
			warnStyle.Println("  First run? Read README.md, then pick 0 to build databases.")
			fmt.Println()

			printH2("Data folders")
			ex := initcheck()
			folders := []string{"Schedules", "Courses", "State", "XML"}
			for i, name := range folders {
				printStatus(ex[i], name)
			}

			warnStyle.Println("  Preparing UI...")
			time.Sleep(4 * time.Second)
			fmt.Println()


			scanner := bufio.NewScanner(os.Stdin)

			for {
				printH1("home")
				printH2("Detected Files", color.New(color.FgGreen))
				printLoaded()
				fmt.Println()
				printH2("Favorites", color.New(color.FgRed))
				printFavs()
				hasDB := hasDatabases()

				fmt.Println()
				printH2("What would you like to do?")
				if !hasDB {
					wrnf("No databases found. Pick 0 first.\n")
				}
				menu := []struct {
					key     string
					label   string
					enabled bool
				}{
					{"0", "Rebuild Databases", true},
					{"1", "Search by Room", hasDB},
					{"2", "Search by Time", hasDB},
					{"3", "Search by Course", hasDB},
					{"4", "Custom SQL Query", hasDB},
					{"5", "Manage Favorites", hasDB},
					{"6", "Settings", true},
					{"7", "About", true},
					{"8", "Exit", true},
				}
				for _, m := range menu {
					if m.enabled {
						printText(fmt.Sprintf("%s. %s", m.key, m.label))
					} else {
						dimStyle.Printf("%s%s. %s\n", indent, m.key, m.label)
					}
				}
				fmt.Println()
				printTextf("Enter your choice (0-8): ")

				if !scanner.Scan() {
					return
				}
				choice := strings.TrimSpace(scanner.Text())
				fmt.Println()

				if !hasDB && choice >= "1" && choice <= "5" {
					wrnf("No databases found. Pick 0 first.\n")
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
					wrnf("Invalid choice %q, please enter 0-8.\n", choice)
				}
			}
		},
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}
