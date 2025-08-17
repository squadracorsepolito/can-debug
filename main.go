package main

import (
	"fmt"
	"log"
	"os"

	"github.com/carolabonamico/can-debug/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// If a DBC file is provided as an argument, load it directly
	var dbcPath string
	if len(os.Args) > 1 && os.Args[1] != "-h" && os.Args[1] != "--help" {
		dbcPath = os.Args[1]

		// Check if the file exists
		if _, err := os.Stat(dbcPath); os.IsNotExist(err) {
			fmt.Printf("Error: File DBC not found: %s\n", dbcPath)
			os.Exit(1)
		}

		fmt.Printf("📁 Loading DBC file: %s\n", dbcPath)
	}

	// Handle help
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		showHelp()
		return
	}

	// Create the initial model
	m := ui.NewModelWithDBC(dbcPath)

	// Start bubbletea
	p := tea.NewProgram(&m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func showHelp() {
	fmt.Print(`🔧 CAN Debug Tool

Use:
  can-debug [file.dbc]  Directly load a DBC file
  can-debug -h|--help   Show this help

Examples:
  can-debug                           # Use the file picker
  can-debug ../../test/server/MCB.dbc # Load specific file

Commands:
  ↑/↓ o k/j    Navigation
  Space        Select/deselect message
  Enter        Confirm/start monitoring
  Tab          Change section
  /            Search (in the message list)
  q            Quit
`)
}
