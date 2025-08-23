package ui

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/squadracorsepolito/acmelib"
	"go.einride.tech/can"
	"go.einride.tech/can/pkg/socketcan"

	canDebug "github.com/squadracorsepolito/can-debug/internal/can"
)

// loadDBC loads the DBC file
func (m *Model) loadDBC() error {
	// Use acmelib to load the DBC file
	file, err := os.Open(m.DBCPath)
	if err != nil {
		return fmt.Errorf("error in opening DCB file: %w", err)
	}
	defer file.Close()

	bus, err := acmelib.ImportDBCFile("debug_bus", file)
	if err != nil {
		return fmt.Errorf("error in loading DCB file: %w", err)
	}

	// Collect all messages from the bus
	m.Messages = make([]*acmelib.Message, 0)
	for _, nodeInt := range bus.NodeInterfaces() {
		m.Messages = append(m.Messages, nodeInt.SentMessages()...)
	}

	m.Decoder = canDebug.NewDecoder(m.Messages)

	return nil
}

// setupMessageList configure the message list
func (m *Model) setupMessageList() {
	items := make([]list.Item, 0, len(m.Messages))

	for _, msg := range m.Messages {
		// Check if this message is selected
		isSelected := false
		for _, selected := range m.SelectedMessages {
			if selected.ID == uint32(msg.GetCANID()) {
				isSelected = true
				break
			}
		}

		canMsg := CANMessage{
			ID:       uint32(msg.GetCANID()),
			Name:     msg.Name(),
			Selected: isSelected,
			Message:  msg,
		}
		items = append(items, canMsg)
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	// Set the width and height of the message list
	width := m.Width
	height := m.Height - 6
	if width == 0 {
		width = 80 // default width
	}
	if height <= 0 {
		height = 15
	}

	m.MessageList = list.New(items, delegate, width, height)
	m.MessageList.SetShowTitle(false)
	m.MessageList.SetShowHelp(false)
	m.MessageList.SetShowStatusBar(false)
	m.MessageList.SetFilteringEnabled(true)
}

// toggleMessageSelection toggles the selection of a message
func (m *Model) toggleMessageSelection() {
	if selectedItem, ok := m.MessageList.SelectedItem().(CANMessage); ok {
		// Check if the message is already selected
		found := false
		for i, msg := range m.SelectedMessages {
			if msg.ID == selectedItem.ID {
				// Remove from selection
				m.SelectedMessages = append(m.SelectedMessages[:i], m.SelectedMessages[i+1:]...)
				found = true
				break
			}
		}

		if !found {
			// Add to selection
			selectedItem.Selected = true
			m.SelectedMessages = append(m.SelectedMessages, selectedItem)
		}

		// Update the list items
		m.updateMessageListItems()
	}
}

// updateMessageListItems updates the list items without rebuilding it
func (m *Model) updateMessageListItems() {
	items := make([]list.Item, 0, len(m.Messages))

	for _, msg := range m.Messages {
		// Check if this message is selected
		isSelected := false
		for _, selected := range m.SelectedMessages {
			if selected.ID == uint32(msg.GetCANID()) {
				isSelected = true
				break
			}
		}

		canMsg := CANMessage{
			ID:       uint32(msg.GetCANID()),
			Name:     msg.Name(),
			Selected: isSelected,
			Message:  msg,
		}
		items = append(items, canMsg)
	}

	// Update the message list with the new items
	currentIndex := m.MessageList.Index()
	m.MessageList.SetItems(items)

	// Maintain the current position in the list
	if currentIndex < len(items) {
		m.MessageList.Select(currentIndex)
	}
}

// setupMonitoringTable configures the monitoring table
func (m *Model) setupMonitoringTable() {
	columns := []table.Column{
		{Title: "Message", Width: 25},
		{Title: "ID", Width: 10},
		{Title: "Signal", Width: 30},
		{Title: "Value", Width: 20},
		{Title: "Raw", Width: 12},
		{Title: "Type", Width: 20},
	}

	rows := []table.Row{}

	m.MonitoringTable = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(m.Height-8),
	)
}

// initializes the table with all signals and value from selected DBC messages
func (m *Model) initializesTableDBCSignals() {
	if len(m.SelectedMessages) == 0 {
		return
	}

	rows := []table.Row{}

	for _, msg := range m.SelectedMessages {
		// Get all signals for this message from the DBC
		signals := msg.Message.Signals()

		if len(signals) == 0 {
			// If no signals, show the message itself
			row := table.Row{
				msg.Name,
				fmt.Sprintf("0x%X", msg.ID),
				"[No signal]",
				"--",
				"--",
				"--",
			}
			rows = append(rows, row)
		} else {
			// Show each signal
			for _, signal := range signals {
				signalInfo := signal.Name()

				// Get signal unit if it's a standard signal
				if stdSignal, err := signal.ToStandard(); err == nil && stdSignal.Unit() != nil {
					signalInfo += fmt.Sprintf(" (%s)", stdSignal.Unit().Name())
				}

				// Show signal bit position and length
				startPos := signal.StartPos()
				size := signal.Size()

				row := table.Row{
					msg.Name,
					fmt.Sprintf("0x%X", msg.ID),
					signalInfo,
					"[In attesa dati]",
					fmt.Sprintf("bit %d:%d", startPos, startPos+size-1),
					m.getSignalTypeString(signal),
				}
				rows = append(rows, row)
			}
		}
	}

	m.MonitoringTable.SetRows(rows)
}

// getSignalTypeString returns a string representation of the signal type
func (m *Model) getSignalTypeString(signal acmelib.Signal) string {
	if stdSignal, err := signal.ToStandard(); err == nil {
		if stdSignal.Type() != nil {
			return fmt.Sprintf("%s (%d bit)", stdSignal.Type().Name(), signal.Size())
		}
		return fmt.Sprintf("standard (%d bit)", signal.Size())
	}

	if _, err := signal.ToEnum(); err == nil {
		return fmt.Sprintf("enum (%d bit)", signal.Size())
	}

	if _, err := signal.ToMuxor(); err == nil {
		return fmt.Sprintf("muxor (%d bit)", signal.Size())
	}

	return fmt.Sprintf("unknown (%d bit)", signal.Size())
}

// this function is intended as a goroutine,
// it will keep receving messages from the Can Network, only saving them when in the "StateMonitoring" State
func (m *Model) startReceavingMessages() {

	if m.CanNetwork == nil {
		// On systems without SocketCAN (like macOS), try to use candump for receiving
		if runtime.GOOS != "linux" {
			m.startReceivingWithCanDump()
			return
		}
		fmt.Println("No SocketCAN connection available - monitoring disabled")
		return
	}

	recv := socketcan.NewReceiver(m.CanNetwork)
	for recv.Receive() {
		if m.State != StateMonitoring {
			break
		}

		frame := recv.Frame()
		decodedSignals := m.Decoder.Decode(context.Background(), frame.ID, frame.Data[:])

		for _, sgn := range decodedSignals {
			m.updateTable(sgn, frame.ID)
		}
	}
	recv.Close()
}

// startReceivingWithCanDump uses candump command to receive CAN messages (for testing on non-Linux)
func (m *Model) startReceivingWithCanDump() {
	// Try to use candump to receive messages
	cmd := exec.Command("candump", "vcan0")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.Err = fmt.Errorf("error creating candump pipe: %v", err)
		return
	}

	if err := cmd.Start(); err != nil {
		m.Err = fmt.Errorf("candump not available: %v (install can-utils or use a Linux system with SocketCAN)", err)
		return
	}

	// Read from candump output in a separate goroutine
	go func() {
		defer cmd.Wait()
		defer stdout.Close()

		buffer := make([]byte, 1024)
		for {
			if m.State != StateMonitoring {
				cmd.Process.Kill()
				break
			}

			n, err := stdout.Read(buffer)
			if err != nil {
				break
			}

			// Parse candump output (format: "vcan0  123   [8]  48 65 6C 6C 6F 00 00 00")
			output := string(buffer[:n])
			m.parseCandumpOutput(output)
		}
	}()
}

// parseCandumpOutput parses the output from candump and updates the table
func (m *Model) parseCandumpOutput(output string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Basic parsing of candump format: "vcan0  123   [8]  48 65 6C 6C 6F 00 00 00"
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		// Extract CAN ID
		canIDStr := parts[1]
		canID, err := strconv.ParseUint(canIDStr, 16, 32)
		if err != nil {
			continue
		}

		// Extract data bytes (skip interface, ID, and length parts)
		dataStart := 3
		for i, part := range parts {
			if strings.HasPrefix(part, "[") {
				dataStart = i + 1
				break
			}
		}

		if dataStart >= len(parts) {
			continue
		}

		// Convert hex bytes to data array
		var data [8]byte
		for i, hexByte := range parts[dataStart:] {
			if i >= 8 {
				break
			}
			if b, err := strconv.ParseUint(hexByte, 16, 8); err == nil {
				data[i] = byte(b)
			}
		}

		// Decode signals using the existing decoder
		decodedSignals := m.Decoder.Decode(context.Background(), uint32(canID), data[:])
		for _, sgn := range decodedSignals {
			m.updateTable(sgn, uint32(canID))
		}
	}
}

// given a signal and its ID, updates the table with the corrisponding value (if the signal it's present in the monitoring table)
func (m *Model) updateTable(sgn *acmelib.SignalDecoding, sgnID uint32) {

	sgnIDhex := fmt.Sprintf("0x%X", sgnID)
	rows := m.MonitoringTable.Rows()
	for i := range rows {
		if rows[i][1] == sgnIDhex && strings.Contains(rows[i][2], sgn.Signal.Name()) {
			rows[i][3] = fmt.Sprintf("%v", sgn.Value)
			break
		}
	}
	m.MonitoringTable.SetRows(rows)
}

// sendMessage sends a CAN message with the provided string content
func (m *Model) sendMessage(message string) {
	// Detect OS and use appropriate method
	if runtime.GOOS == "linux" && m.CanNetwork != nil {
		// Use SocketCAN on Linux
		m.sendWithSocketCAN(message)
	} else {
		// Use cansend command for testing on other OS
		m.sendWithCanSend(message)
	}
}

// sendWithSocketCAN sends a message using SocketCAN (Linux)
func (m *Model) sendWithSocketCAN(message string) {
	// Convert the string to hex bytes
	data := []byte(message)
	if len(data) > 8 {
		data = data[:8] // CAN frames can have max 8 bytes of data
	}

	// Use a default CAN ID for sent messages
	frame := can.Frame{
		ID:     0x123, // Default CAN ID for sent messages
		Data:   [8]byte{},
		Length: uint8(len(data)),
	}

	// Copy data to frame
	copy(frame.Data[:], data)

	// Send the frame
	transmitter := socketcan.NewTransmitter(m.CanNetwork)
	err := transmitter.TransmitFrame(context.Background(), frame)
	if err != nil {
		m.SendStatus = fmt.Sprintf("⚠️ SocketCAN error: %v", err)
		m.LastSentMessage = "" // Clear last sent message on error
	} else {
		m.SendStatus = fmt.Sprintf("📡 Sent via SocketCAN (ID: 0x%X)", frame.ID)
		m.LastSentMessage = message // Set last sent message only on success
	}
}

// sendWithCanSend sends a message using cansend command (for testing)
func (m *Model) sendWithCanSend(message string) {
	// Convert string to hex format
	data := []byte(message)
	if len(data) > 8 {
		data = data[:8] // CAN frames can have max 8 bytes of data
	}

	// Build hex string
	hexData := hex.EncodeToString(data)

	// Pad with zeros to make it 8 bytes (16 hex chars)
	for len(hexData) < 16 {
		hexData += "00"
	}

	// Default interface name
	interfaceName := "vcan0"

	// Build cansend command: cansend vcan0 123#hexdata
	canID := "123"
	canMessage := canID + "#" + hexData

	// Execute cansend command
	cmd := exec.Command("cansend", interfaceName, canMessage)
	err := cmd.Run()

	if err != nil {
		// Set error status instead of printing to console
		m.SendStatus = fmt.Sprintf("⚠️ Error: %v (install can-utils)", err)
		m.LastSentMessage = "" // Clear last sent message on error
	} else {
		// Set success status
		m.SendStatus = fmt.Sprintf("📡 Sent via %s (ID: %s)", interfaceName, canID)
		m.LastSentMessage = message // Set last sent message only on success
	}
}
