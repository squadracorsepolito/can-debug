package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/squadracorsepolito/can-debug/internal/ui"
	"go.einride.tech/can/pkg/socketcan"
)

func main() {
	// Handle help
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		showHelp()
		return
	}
	
	//connecting and setting up the can network
	var canNetworkName string
	if len(os.Args) <= 1 {
		fmt.Print("Warning: no name for the can network was provided\n")
		return   //canNetworkName = ""     <---------------------------------   replace this line for debug TUI
	}else{
		canNetworkName = os.Args[1]
	}
	var conn net.Conn
	// Try to connect to SocketCAN
	c, err := socketcan.DialContext(context.Background(), "can", canNetworkName)
	if err != nil {
		fmt.Printf("Warning: Error opening SocketCAN on Linux: %v\n", err)
		return   //conn = nil               <---------------------------------   replace this line for debug TUI
	} else {
		conn = c
		defer conn.Close()
	}
	

	// If a DBC file is provided as an argument, load it directly
	var dbcPath string
	if len(os.Args) > 2 {
		dbcPath = os.Args[2]

		// Check if the file exists
		if _, err := os.Stat(dbcPath); os.IsNotExist(err) {
			fmt.Printf("Error: File DBC not found: %s\n", dbcPath)
			os.Exit(0)
		}

		fmt.Printf("📁 Loading DBC file: %s\n", dbcPath)
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
  can-debug [name_of_can_network] -> specify the CAN network name and load a dbc file with the file picker
  can-debug [name_of_can_network] [file.dbc] -> Directly load a DBC file
  can-debug -h|--help   Show this help

Examples:
  can-debug                              # Visualize only the interface(no real can network) + Use the file picker
  can-debug vcan0                        # Try to connect to the can network "vcan0" + Use the file picker
  can-debug vcan0 internal/test/MCB.dbc  # Try to connect to the can network "vcan0" + Load specific file

Platform Support:
  You can only transmit CAN messages on Linux systems, and you need to create the can network beforehand (see README file).

Navigation Commands:
  ↑/↓ or k/j   Navigate up/down in lists and tables
  Tab          Go back to previous screen
  q            Quit application

File Selection:
  Enter        Open directory or select .dbc file

Send/Receive Mode Selection:
  ↑/↓          Choose between Send or Receive mode
  Enter        Confirm selection

Message List:
  /            Search messages
  Space        Select/deselect message for monitoring or sending
  Enter        Confirm selection and proceed

Send Configuration:
  Navigation:  ↑/↓ move between signals • Tab go back • q quit
  Action:      Enter send once • Space toggle continuous sending
               ←→ adjust cycle time • s stop all signals

  Individual Message Control:
    - Use ←→ arrows to adjust cycle time (10ms increments, range: 10ms-10s) (can be changed by changing the value rangeMs in internal/ui/types.go)
    - Enter: Send signal once (single shot)
    - Space: Toggle continuous sending at set frequency
    - s: Emergency stop all continuous signals
    - Input field: Enter signal value (supports decimals, negatives)

Receive Mode (Monitoring):
  Real-time monitoring of selected CAN messages with signal decoding
`)
}
