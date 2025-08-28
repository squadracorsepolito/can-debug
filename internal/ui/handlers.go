package ui

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"strconv"
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

	// Initialize the MessageList immediately after loading messages
	// This prevents null pointer issues when switching between send/receive modes
	m.setupMessageList()

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

		//check if the message is being sent
		cycleMessage := ""
		mex, ok := m.ActiveMessages[int(msg.GetCANID())]
		if ok {
			cycleMessage = fmt.Sprintf(" - (Currently being send every %dms)", mex.frequency)
		}

		canMsg := CANMessage{
			ID:       uint32(msg.GetCANID()),
			Name:     fmt.Sprint(msg.Name(), cycleMessage),
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
		if m.SendReceiveChoice == 0 {
			// Send mode - solo un messaggio alla volta
			m.SelectedMessages = []CANMessage{} // Clear all selections first
			selectedItem.Selected = true
			m.SelectedMessages = append(m.SelectedMessages, selectedItem)
		} else {
			// Receive mode - selezione multipla come prima
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
		}

		// Update the list items
		m.updateMessageListItems()
	}
}

// updateMessageListItems updates the list items without rebuilding it
func (m *Model) updateMessageListItems() {
	// Check if MessageList has been initialized
	if m.MessageList.Items() == nil {
		return
	}

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

		//check if the message is being sent
		cycleMessage := ""
		mex, ok := m.ActiveMessages[int(msg.GetCANID())]
		if ok {
			cycleMessage = fmt.Sprintf(" - (Currently being send every %dms)", mex.frequency)
		}

		canMsg := CANMessage{
			ID:       uint32(msg.GetCANID()),
			Name:     fmt.Sprint(msg.Name(), cycleMessage),
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
		{Title: "Signal", Width: 35},
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
// qui !
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
		m.Err = fmt.Errorf("no SocketCAN connection available - monitoring disabled")
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

// setupSendConfiguration prepares the send configuration table and signals
func (m *Model) setupSendConfiguration() {
	m.SendSignals = make([]SendSignal, 0)

	// Create send signals from selected messages
	m.CycleTime = rangeMs
	msg := m.SelectedMessages[0]
	for _, signal := range msg.Message.Signals() {
		sendSignal := SendSignal{
			SignalName: signal.Name(),
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
		{Title: "Signal", Width: 35},
		{Title: "Cycle(ms)", Width: 10},
		{Title: "Status", Width: 15},
		{Title: "Value", Width: 25},
	}

	var status string
	_, ok := m.ActiveMessages[int(m.SelectedMessages[0].ID)]
	if ok {
		status = "▶️  sending"
	} else {
		status = "⏸️  stopped"
	}

	rows := make([]table.Row, len(m.SendSignals))
	for i, signal := range m.SendSignals {
		signalWithUnit := signal.SignalName
		if signal.Unit != "" {
			signalWithUnit += " (" + signal.Unit + ")"
		}

		// Format cycle time with padding for alignment - show "-" for single shot
		var cycleStr string
		if signal.IsSingleShot {
			cycleStr = fmt.Sprintf("%-8s", "-")
		} else {
			cycleStr = fmt.Sprintf("%-8d", m.CycleTime)
		}

		// Format status with padding for alignment
		statusStr := fmt.Sprintf("%-8s", status)

		rows[i] = table.Row{
			m.SelectedMessages[0].Name,
			fmt.Sprintf("0x%-6X", m.SelectedMessages[0].ID), // Left-aligned with padding
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

	var status string
	_, ok := m.ActiveMessages[int(m.SelectedMessages[0].ID)]
	if ok {
		status = "▶️  sending"
	} else {
		status = "⏸️  stopped"
	}

	for i, signal := range m.SendSignals {
		signalWithUnit := signal.SignalName
		if signal.Unit != "" {
			signalWithUnit += " (" + signal.Unit + ")"
		}

		// Format cycle time with padding for alignment - show "-" for single shot
		var cycleStr string
		if signal.IsSingleShot {
			cycleStr = fmt.Sprintf("%-8s", "-")
		} else {
			cycleStr = fmt.Sprintf("%-8d", m.CycleTime)
		}

		// Format status with padding for alignment
		statusStr := fmt.Sprintf("%-8s", status)

		rows[i] = table.Row{
			m.SelectedMessages[0].Name,
			fmt.Sprintf("0x%-6X", m.SelectedMessages[0].ID), // Left-aligned with padding
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

//m.SendStatus = fmt.Sprintf("✅  Sent signals: %s", strings.Join(signalNames, ", "))


// This starts a goroutine to send the current selected message cyclically
func (m *Model) startCyclicalSending() {
	
	frame, ok := m.GenarateFrame()
	if !ok {
		return // Error message already set in GenarateFrame
	}
	
	//try to send the first frame immediately to catch errors before starting the goroutine
	err := m.sendFrame(frame)
	if err != nil {
		m.SendStatus = fmt.Sprintf("⚠️ SocketCAN error: %v", err)
		return
	} 

	//build info for stopping the message
	ctx, cancel := context.WithCancel(context.Background())
	mex := infoSending{
		stop:      cancel,
		frequency: m.CycleTime,
	}
	m.ActiveMessages[int(frame.ID)] = mex

	//this goroutine sends a message every 'interval' of time, ctx is used to stop
	go func(interval time.Duration, ctx context.Context, frame can.Frame) {
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				m.sendFrame(frame)
			}
		}
	}(time.Duration(mex.frequency)*time.Millisecond, ctx, frame)

	m.SendStatus = fmt.Sprintf("🔄  Message '%s': Cyclical sending started (interval: %dms).", m.SelectedMessages[0].Name, m.CycleTime)
	// Update the table to reflect the new status
	m.updateSendTableRows()
}

// stopCyclicalSending stops cyclical sending of the current selected message
func (m *Model) stopCyclicalSending() {
	messageID := m.SelectedMessages[0].ID //ID in decimal
	messageName := m.SelectedMessages[0].Name

	//stop goroutine that is sending the message
	mex := m.ActiveMessages[int(messageID)]
	mex.stop()
	delete(m.ActiveMessages, int(messageID))
	m.SendStatus = fmt.Sprintf("🛑 Message '%s': Cyclical sending stopped", messageName)
	m.updateSendTableRows()
}

func (m *Model) stopAllMessages() {
	for id, mex := range m.ActiveMessages {
		mex.stop()
		delete(m.ActiveMessages, id)
	}
	m.SendStatus = "🛑ALL🛑 cyclical sending stopped"
	m.updateSendTableRows()
}

// sendSingleMessage sends all signals of a message once
func (m *Model) sendSingleMessage() {
	// Generate and send the CAN frame
	frame, ok := m.GenarateFrame()
	if !ok {
		return // Error message already set in GenarateFrame
	}
	err := m.sendFrame(frame)
	if err != nil {
		m.SendStatus = fmt.Sprintf("⚠️ SocketCAN error: %v", err)
		return
	} 

	// Find all signals for this message and send them
	sentCount := 0
	for i := range m.SendSignals {
		signal := &m.SendSignals[i]
		signal.IsSingleShot = true
		sentCount++
	}
	m.SendStatus = fmt.Sprintf("📤 Sent message '%s' (%d signals) once: %v", m.SelectedMessages[0].Name, sentCount, frame.Data)
	
	// Update display and reset single shot flags after a brief moment (blink effect)
	m.updateSendTableRows()
	go func() {
		time.Sleep(2 * time.Second)
		for i := range m.SendSignals {
			m.SendSignals[i].IsSingleShot = false
		}
		m.updateSendTableRows()
	}()
}

// adjustMessageCycleTime adjusts cycle time for all signals of a message
func (m *Model) adjustMessageCycleTime(delta int) {
	// Calculate new cycle time with bounds checking
	newCycleTime := m.CycleTime + delta
	if newCycleTime < rangeMs {
		newCycleTime = rangeMs // rangeMs is a constant that rapresents the minimum interval for the cycle
	}
	if newCycleTime > 10000 {
		newCycleTime = 10000 // max 10 seconds
	}

	m.CycleTime = newCycleTime
	// Update display
	m.updateSendTableRows()

	// Show status
	m.SendStatus = fmt.Sprintf("🔄 Set cycle time to %dms for message '%s'", newCycleTime, m.SelectedMessages[0].Name)
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

// GenarateFrame creates a CAN frame from the current SendSignals values
func (m *Model) GenarateFrame() (can.Frame, bool) {
	frame := can.Frame{}
	mex := m.SelectedMessages[0].Message
	
	// for each signal
	for _, signal := range mex.Signals() {
		//find inserted value
		value, err := m.getInsertedValue(signal)
		if err != nil {
			m.SendStatus = fmt.Sprintf("⚠️  Error getting signal %s: %s", signal.Name(), err.Error())
			return frame, false
		}

		//update the value in the signal
		switch signal.Kind().String(){
		case "standard":
			standardSign, _ := signal.ToStandard()
			err = standardSign.UpdateEncodedValue(value)
			if err != nil {
				m.SendStatus = fmt.Sprintf("⚠️  Error encoding signal '%s': value must be between %v and %v", signal.Name(), standardSign.Type().Min(), standardSign.Type().Max())
				return frame, false
			}
		case "enum":
			enumSign, _ := signal.ToEnum()
			err = enumSign.UpdateEncodedValue(int(value))
			if err != nil {
				m.SendStatus = fmt.Sprintf("⚠️  Error encoding signal '%s'(type enum): value %d is not valid", signal.Name(), int(value))
				return frame, false
			}
		case "muxor":
			muxorSign, _ := signal.ToMuxor()
			err = muxorSign.UpdateEncodedValue(int(value))
			if err != nil {
				m.SendStatus = fmt.Sprintf("⚠️  Error encoding signal '%s'(type muxor): value %d is out of bounds", signal.Name(), int(value))
				return frame, false
			}
		}
	}

	//build and return the frame
	data := mex.SignalLayout().Encode()
	copy(frame.Data[:], data)
	frame.ID = uint32(mex.GetCANID())
	frame.Length = uint8(len(frame.Data))

	return frame, true
}

// getInsertedValue cheks if the signal passed is currently selected, if it is it return the value inserted in input
func (m *Model) getInsertedValue(signal acmelib.Signal) (float64, error){

	for _, SendSig := range m.SendSignals{
		if SendSig.SignalName == signal.Name(){
			value := SendSig.TextInput.Value()
			//if value is empty assign 0
			if value == ""{
				return 0, nil
			}
			return strconv.ParseFloat(value, 64) 
		}
	}

	return 0, fmt.Errorf("segnale non trovato")
}

// sendWithSocketCAN sends a message using SocketCAN (Linux)
func (m *Model) sendFrame(frame can.Frame) error{
	// Send the frame
	if m.CanNetwork == nil {
		return fmt.Errorf("⚠️  No SocketCAN transmitter available")
	}
	return m.Transmitter.TransmitFrame(context.Background(), frame)
}