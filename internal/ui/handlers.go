package ui

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/squadracorsepolito/acmelib"
	"go.einride.tech/can"
	"go.einride.tech/can/pkg/socketcan"

	canDebug "github.com/squadracorsepolito/can-debug/internal/can"
)

// validateDecimalInput validates that input contains only decimal numbers (including negative)
func validateDecimalInput(s string) error {
	if s == "" || s == "-" || s == "." || s == "-." {
		return nil // Allow empty string, single minus, single dot, and combination for partial input
	}
	// Allow decimal numbers (positive and negative) with optional decimal point
	// Supports formats: 123, -123, 123.45, -123.45, .5, -.5, 0.5, -0.5
	matched, _ := regexp.MatchString(`^-?(\d+\.?\d*|\.\d+)$`, s)
	if !matched {
		return fmt.Errorf("only decimal numbers allowed (e.g., 123, -45.67, .5)")
	}
	return nil
}

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
		{Title: "ID", Width: 8},
		{Title: "Signal", Width: 30},
		{Title: "Value", Width: 20},
		{Title: "Raw", Width: 15},
		{Title: "Type", Width: 20},
	}

	rows := []table.Row{}

	m.MonitoringTable = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(m.Height-8),
	)

	// Ensure the table is properly focused
	m.MonitoringTable.Focus()
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

// setupSendConfiguration prepares the send configuration table and signals
func (m *Model) setupSendConfiguration() {
	m.SendSignals = make([]SendSignal, 0)

	// Create send signals from selected messages
	for _, msg := range m.SelectedMessages {
		for _, signal := range msg.Message.Signals() {
			sendSignal := SendSignal{
				MessageName: msg.Name,
				ID:          msg.ID,
				SignalName:  signal.Name(),
				Value:       "0",   // default value
				CycleTime:   100,   // default 100ms
				IsActive:    false, // initially stopped
				TaskID:      0,     // will be assigned when started
			}

			// Extract unit information from the signal
			if stdSignal, err := signal.ToStandard(); err == nil && stdSignal.Unit() != nil {
				sendSignal.Unit = stdSignal.Unit().Name()
			}

			// Create text input for this signal
			ti := textinput.New()
			ti.Placeholder = "0"
			ti.CharLimit = 20
			ti.Width = 15
			// Set validation function for decimal numbers
			ti.Validate = validateDecimalInput
			sendSignal.TextInput = ti

			m.SendSignals = append(m.SendSignals, sendSignal)
		}
	}

	// Setup the send table
	m.setupSendTable()

	// Focus the first input if available
	if len(m.SendSignals) > 0 {
		m.CurrentInputIndex = 0
		m.SendSignals[0].TextInput.Focus()
		m.SendTable.SetCursor(0) // Set cursor to first row
		// Make sure all other inputs are blurred
		for i := 1; i < len(m.SendSignals); i++ {
			m.SendSignals[i].TextInput.Blur()
		}
	}
}

// setupSendTable configures the table for send configuration
func (m *Model) setupSendTable() {
	columns := []table.Column{
		{Title: "Message", Width: 25},
		{Title: "ID", Width: 8},
		{Title: "Signal", Width: 30},
		{Title: "Cycle(ms)", Width: 10},
		{Title: "Status", Width: 10},
		{Title: "Value", Width: 25},
	}

	rows := make([]table.Row, len(m.SendSignals))
	for i, signal := range m.SendSignals {
		signalWithUnit := signal.SignalName
		if signal.Unit != "" {
			signalWithUnit += " (" + signal.Unit + ")"
		}

		status := "⏸️" // paused/stopped
		if signal.IsActive {
			status = "▶️" // playing/active
		}

		// Format cycle time with padding for alignment - show "-" for single shot
		var cycleStr string
		if signal.IsSingleShot {
			cycleStr = fmt.Sprintf("%-8s", "-")
		} else {
			cycleStr = fmt.Sprintf("%-8d", signal.CycleTime)
		}

		// Format status with padding for alignment
		statusStr := fmt.Sprintf("%-8s", status)

		rows[i] = table.Row{
			signal.MessageName,
			fmt.Sprintf("0x%-6X", signal.ID), // Left-aligned with padding
			signalWithUnit,
			cycleStr,
			statusStr,
			signal.TextInput.View(),
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	m.SendTable = t

	// Ensure the table is properly focused and cursor is visible
	m.SendTable.Focus()
	if len(m.SendSignals) > 0 {
		m.SendTable.SetCursor(0)
	}
}

// updateSendTableRows updates the send table with current input values
func (m *Model) updateSendTableRows() {
	rows := make([]table.Row, len(m.SendSignals))
	for i, signal := range m.SendSignals {
		signalWithUnit := signal.SignalName
		if signal.Unit != "" {
			signalWithUnit += " (" + signal.Unit + ")"
		}

		status := "⏸️" // paused/stopped
		if signal.IsActive {
			status = "▶️" // playing/active
		}

		// Format cycle time with padding for alignment - show "-" for single shot
		var cycleStr string
		if signal.IsSingleShot {
			cycleStr = fmt.Sprintf("%-8s", "-")
		} else {
			cycleStr = fmt.Sprintf("%-8d", signal.CycleTime)
		}

		// Format status with padding for alignment
		statusStr := fmt.Sprintf("%-8s", status)

		rows[i] = table.Row{
			signal.MessageName,
			fmt.Sprintf("0x%-6X", signal.ID), // Left-aligned with padding
			signalWithUnit,
			cycleStr,
			statusStr,
			signal.TextInput.View(),
		}
	}
	m.SendTable.SetRows(rows)

	// Set cursor to the current input index to highlight the correct row
	if m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
		m.SendTable.SetCursor(m.CurrentInputIndex)
	}

	// Ensure cursor visibility
	m.ensureTableCursorVisible()
}

// sendConfiguredSignals sends the configured signals via CAN
func (m *Model) sendConfiguredSignals() {
	if len(m.SendSignals) == 0 {
		m.SendStatus = "No signals configured to send"
		return
	}

	// Validate that all required values are entered
	for _, signal := range m.SendSignals {
		if signal.TextInput.Value() == "" {
			m.SendStatus = fmt.Sprintf("Please enter a value for signal '%s'", signal.SignalName)
			return
		}
	}

	// TODO: Implement actual CAN sending logic
	// For now, just show success message
	var signalNames []string
	for _, signal := range m.SendSignals {
		signalNames = append(signalNames, fmt.Sprintf("%s=%s", signal.SignalName, signal.TextInput.Value()))
	}

	m.SendStatus = fmt.Sprintf("✅ Sent signals: %s", strings.Join(signalNames, ", "))
}

// startCyclicalSending starts sending signals cyclically
func (m *Model) startCyclicalSending() {
	if len(m.SendSignals) == 0 {
		m.SendStatus = "No signals configured to send"
		return
	}

	// Validate that all required values are entered
	for _, signal := range m.SendSignals {
		if signal.TextInput.Value() == "" {
			m.SendStatus = fmt.Sprintf("Please enter a value for signal '%s'", signal.SignalName)
			return
		}
	}

	m.IsSendingCyclical = true
	m.SendStatus = fmt.Sprintf("🔄 Cyclical sending started (interval: %dms). Press Enter to stop.", m.SendInterval)

	// TODO: Implement actual cyclical CAN sending logic
	// For now, just show status message
}

// stopCyclicalSending stops cyclical sending of signals
func (m *Model) stopCyclicalSending() {
	m.IsSendingCyclical = false
	m.SendStatus = "⏹️ Cyclical sending stopped"

	// TODO: Implement actual stopping of cyclical CAN sending
}

// toggleMessageSending toggles the active state of all signals in a message
func (m *Model) toggleMessageSending(index int) {
	if index >= 0 && index < len(m.SendSignals) {
		currentSignal := &m.SendSignals[index]
		messageID := currentSignal.ID
		messageName := currentSignal.MessageName

		// Determine if any signal in this message is currently active
		anyActive := false
		for i := range m.SendSignals {
			if m.SendSignals[i].ID == messageID && m.SendSignals[i].IsActive {
				anyActive = true
				break
			}
		}

		// Toggle all signals in this message to the opposite state
		newState := !anyActive
		activeCount := 0

		for i := range m.SendSignals {
			signal := &m.SendSignals[i]
			if signal.ID == messageID {
				// Validate that the value is entered if starting
				if newState && signal.TextInput.Value() == "" {
					m.SendStatus = fmt.Sprintf("Please enter a value for signal '%s' in message '%s'", signal.SignalName, messageName)
					return
				}

				signal.IsActive = newState

				if signal.IsActive {
					// Start sending this signal individually
					m.startIndividualSignalSending(i)
					activeCount++
				} else {
					// Stop sending this signal
					m.stopIndividualSignalSending(i)
				}
			}
		}

		// Update the table to reflect the new status
		m.updateSendTableRows()

		// Show status message
		if newState {
			m.SendStatus = fmt.Sprintf("▶️ Started continuous sending of message '%s' (%d signals)", messageName, activeCount)
		} else {
			m.SendStatus = fmt.Sprintf("⏸️ Stopped sending message '%s'", messageName)
		}
	}
}

// toggleSignalSending toggles the active state of a specific signal
func (m *Model) toggleSignalSending(index int) {
	if index >= 0 && index < len(m.SendSignals) {
		signal := &m.SendSignals[index]
		signal.IsActive = !signal.IsActive

		if signal.IsActive {
			// Start sending this signal individually
			m.startIndividualSignalSending(index)
		} else {
			// Stop sending this signal
			m.stopIndividualSignalSending(index)
		}

		// Update the table to reflect the new status
		m.updateSendTableRows()
	}
}

// setCycleTimePrompt sets up a prompt for editing cycle time (simplified version)
func (m *Model) setCycleTimePrompt(index int) {
	if index >= 0 && index < len(m.SendSignals) {
		signal := &m.SendSignals[index]
		// For now, just increment by 100ms as a simple implementation
		// In future, this could open a dedicated input dialog
		if signal.CycleTime < 10000 {
			signal.CycleTime += 100
		} else {
			signal.CycleTime = 100 // Reset to minimum
		}
		m.updateSendTableRows()
	}
}

// startIndividualSignalSending starts sending a specific signal with its own frequency
func (m *Model) startIndividualSignalSending(index int) {
	if index >= 0 && index < len(m.SendSignals) {
		signal := &m.SendSignals[index]

		// Check if this signal is already being sent
		if _, exists := m.ActiveTasks[signal.TaskID]; exists {
			// Already active, no need to start again
			return
		}

		// Create a new task ID if needed
		if signal.TaskID == 0 {
			signal.TaskID = m.NextTaskID
			m.NextTaskID++
		}

		// Create a stop channel for this task
		stopChan := make(chan struct{})
		m.ActiveTasks[signal.TaskID] = stopChan

		// Start a goroutine for this signal
		go func(sig SendSignal, stop chan struct{}) {
			ticker := time.NewTicker(time.Duration(sig.CycleTime) * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					// TODO: Send the actual CAN message here
					// For now, just a placeholder
				}
			}
		}(*signal, stopChan)

		m.SendStatus = fmt.Sprintf("▶️ Started sending %s every %dms", signal.SignalName, signal.CycleTime)
	}
}

// stopIndividualSignalSending stops sending a specific signal
func (m *Model) stopIndividualSignalSending(index int) {
	if index >= 0 && index < len(m.SendSignals) {
		signal := &m.SendSignals[index]

		// Check if this signal has an active task
		if stopChan, exists := m.ActiveTasks[signal.TaskID]; exists {
			// Send stop signal
			close(stopChan)
			// Remove from active tasks
			delete(m.ActiveTasks, signal.TaskID)

			m.SendStatus = fmt.Sprintf("⏸️ Stopped sending %s", signal.SignalName)
		}
	}
}

// sendSingleMessage sends all signals of a message once
func (m *Model) sendSingleMessage(index int) {
	if index >= 0 && index < len(m.SendSignals) {
		currentSignal := &m.SendSignals[index]
		messageID := currentSignal.ID
		messageName := currentSignal.MessageName

		// Find all signals for this message and send them
		sentCount := 0
		for i := range m.SendSignals {
			signal := &m.SendSignals[i]
			if signal.ID == messageID {
				// Validate that the value is entered
				if signal.TextInput.Value() == "" {
					m.SendStatus = fmt.Sprintf("Please enter a value for signal '%s' in message '%s'", signal.SignalName, messageName)
					return
				}

				// Mark as single shot and count
				signal.IsSingleShot = true
				sentCount++
			}
		}

		// Update display
		m.updateSendTableRows()

		// TODO: Implement actual CAN sending logic here
		// For now, just show success message
		m.SendStatus = fmt.Sprintf("📤 Sent message '%s' (%d signals) once", messageName, sentCount)

		// Reset single shot flags after a brief moment
		go func() {
			time.Sleep(2 * time.Second)
			for i := range m.SendSignals {
				signal := &m.SendSignals[i]
				if signal.ID == messageID {
					signal.IsSingleShot = false
				}
			}
			m.updateSendTableRows()
		}()
	}
}

// adjustMessageCycleTime adjusts cycle time for all signals of a message
func (m *Model) adjustMessageCycleTime(index int, delta int) {
	if index >= 0 && index < len(m.SendSignals) {
		currentSignal := &m.SendSignals[index]
		messageID := currentSignal.ID
		messageName := currentSignal.MessageName

		// Calculate new cycle time with bounds checking
		newCycleTime := currentSignal.CycleTime + delta
		if newCycleTime < 50 {
			newCycleTime = 50 // min 50ms
		}
		if newCycleTime > 10000 {
			newCycleTime = 10000 // max 10 seconds
		}

		// Apply to all signals of this message
		for i := range m.SendSignals {
			signal := &m.SendSignals[i]
			if signal.ID == messageID {
				signal.CycleTime = newCycleTime
				signal.IsSingleShot = false // Reset single shot when adjusting cycle
			}
		}

		// Update display
		m.updateSendTableRows()

		// Show status
		m.SendStatus = fmt.Sprintf("🔄 Set cycle time to %dms for message '%s'", newCycleTime, messageName)
	}
}

// sendSingleSignal sends a single signal once
func (m *Model) sendSingleSignal(index int) {
	if index >= 0 && index < len(m.SendSignals) {
		signal := &m.SendSignals[index]

		// Validate that the value is entered
		if signal.TextInput.Value() == "" {
			m.SendStatus = fmt.Sprintf("Please enter a value for signal '%s'", signal.SignalName)
			return
		}

		// Mark as single shot and update display
		signal.IsSingleShot = true
		m.updateSendTableRows()

		// TODO: Implement actual CAN sending logic here
		// For now, just show success message
		m.SendStatus = fmt.Sprintf("📤 Sent %s = %s once", signal.SignalName, signal.TextInput.Value())

		// Reset single shot flag after a brief moment
		go func() {
			time.Sleep(2 * time.Second)
			signal.IsSingleShot = false
			m.updateSendTableRows()
		}()
	}
}

// ensureTableCursorVisible ensures the table cursor remains visible during scroll
func (m *Model) ensureTableCursorVisible() {
	// For send table
	if m.State == StateSendConfiguration && m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
		// Force the table to focus and set cursor position
		m.SendTable.Focus()
		m.SendTable.SetCursor(m.CurrentInputIndex)
	}

	// For monitoring table
	if m.State == StateMonitoring {
		// Ensure monitoring table is focused
		m.MonitoringTable.Focus()
	}
}

// stopAllSignalSending stops all active signal sending tasks
func (m *Model) stopAllSignalSending() {
	// Stop all active tasks
	for taskID, stopChan := range m.ActiveTasks {
		close(stopChan)
		delete(m.ActiveTasks, taskID)
	}

	// Reset all signals to inactive
	for i := range m.SendSignals {
		m.SendSignals[i].IsActive = false
		m.SendSignals[i].IsSingleShot = false
	}

	// Update table display
	m.updateSendTableRows()

	m.SendStatus = "🛑 All signal sending stopped"
}
