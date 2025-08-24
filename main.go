package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/squadracorsepolito/can-debug/internal/ui"
	"go.einride.tech/can/pkg/socketcan"
)

func main() {
	// If a DBC file is provided as an argument, load it directly
	var dbcPath string
	if len(os.Args) > 1 && os.Args[1] != "-h" && os.Args[1] != "--help" {
		dbcPath = os.Args[1]

		// Check if the file exists
		if _, err := os.Stat(dbcPath); os.IsNotExist(err) {
			fmt.Printf("Error: File DBC not found: %s\n", dbcPath)
			os.Exit(0)
		}

		fmt.Printf("📁 Loading DBC file: %s\n", dbcPath)
	}

	//connecting and setting up the can network
	var conn net.Conn
	if runtime.GOOS == "linux" {
		// Try to connect to SocketCAN on Linux
		c, err := socketcan.DialContext(context.Background(), "can", "vcan0")
		if err != nil {
			fmt.Printf("Warning: Error opening SocketCAN on Linux: %v\n", err)
			fmt.Printf("Continuing in test mode (will use cansend for sending)\n")
			conn = nil
		} else {
			conn = c
			defer conn.Close()
		}
	} else {
		// On non-Linux systems, continue without SocketCAN
		fmt.Printf("Running on %s - no possible transmission\n", runtime.GOOS)
		conn = nil
	}

	// Handle help
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		showHelp()
		return
	}

	// Create the initial model
	m := ui.NewModelWithDBC(dbcPath, conn)

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

Platform Support:
  Linux:   Uses SocketCAN (vcan0) for sending and receiving
  macOS:   Uses cansend command for sending (receiving disabled)
           Make sure can-utils is installed and vcan0 is configured

Commands:
  ↑/↓ o k/j    Navigation
  Space        Select/deselect message
  Enter        Confirm/start monitoring
  Tab          Change section/go back
  /            Search (in the message list)
  q            Quit

Send Mode:
  Enter your message and press Enter to send via CAN
  Tab to return to mode selection
`)
}
