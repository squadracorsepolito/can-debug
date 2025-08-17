package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the current state of the UI
func (m Model) View() string {
	switch m.State {
	case StateFilePicker:
		return m.filePickerView()
	case StateMessageSelector:
		return m.messageSelectorView()
	case StateMonitoring:
		return m.monitoringView()
	default:
		return "Not recognized state"
	}
}

// filePickerView renders the file picker view
func (m Model) filePickerView() string {
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("🔧 CAN Debug Tool - Select a DBC file"))
	s.WriteString("\n\n")

	s.WriteString("↑/k up • ↓/j down • Enter: open directory/select .dbc file • q quit")
	s.WriteString("\n\n")

	if m.Err != nil {
		s.WriteString(fmt.Sprintf("Error: %v", m.Err))
		s.WriteString("\n\n")
	}

	s.WriteString(m.FilePicker.View())

	return s.String()
}

// messageSelectorView renders the message selector view
func (m Model) messageSelectorView() string {
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("📋 Select CAN messages to be monitored"))
	s.WriteString(fmt.Sprintf(" (File: %s)", m.DBCPath))
	s.WriteString("\n\n")

	// Display the message list
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("📋 Select signals to be displayed:"))
	s.WriteString("\n\n")

	// Status bar with commands for the message list
	s.WriteString("↑/k up • ↓/j down • / filter • Space select/deselect • Enter start monitoring • Tab change section • q quit • ? more")
	s.WriteString("\n\n")

	s.WriteString(m.MessageList.View())

	return s.String()
}

// monitoringView renders the monitoring view
func (m Model) monitoringView() string {
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("📡 Real time CAN monitoring"))
	s.WriteString(fmt.Sprintf(" | Last update: %s", m.LastUpdate.Format("15:04:05.000")))
	s.WriteString("\n\n")

	// Status bar with commands for the monitoring table
	s.WriteString("↑/k up • ↓/j down • Tab back to selection • q quit")
	s.WriteString("\n\n")

	s.WriteString(m.MonitoringTable.View())

	return s.String()
}
