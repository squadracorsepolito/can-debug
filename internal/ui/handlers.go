package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/squadracorsepolito/acmelib"

	"github.com/carolabonamico/can-debug/internal/can"
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

	m.Decoder = can.NewDecoder(m.Messages)

	return nil
}

// setupMessageList confgure the message list
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
	height := m.Height - 4
	if width == 0 {
		width = 80 // default width
	}
	if height <= 0 {
		height = 20 // default height
	}

	m.MessageList = list.New(items, delegate, width, height)
	m.MessageList.Title = "Available CAN Messages"
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

// showDBCSignals shows all signals from selected DBC messages
func (m *Model) showDBCSignals() {
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
	m.LastUpdate = time.Now()
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
