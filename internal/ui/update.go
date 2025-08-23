package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// Update updates the model based on the received message
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		// Update the height of the file picker based on the window size
		switch m.State {
		case StateFilePicker:
			m.FilePicker.Height = msg.Height - 4
		case StateSendReceiveSelector:
			// No handling needed
		case StateMessageSelector:
			m.MessageList.SetWidth(msg.Width)
			m.MessageList.SetHeight(msg.Height - 6)
		case StateMonitoring:
			m.MonitoringTable.SetWidth(msg.Width)
			m.MonitoringTable.SetHeight(msg.Height - 4)
		case StateSending:
			m.TextInput.Width = msg.Width - 4
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":

			switch m.State {
			case StateSendReceiveSelector:
				// From send/receive selector, go to the appropriate next state
				if m.SendReceiveChoice == 0 {
					// Send mode
					m.State = StateSending
					m.TextInput.Focus()
				} else {
					// Receive mode
					m.State = StateMessageSelector
					m.setupMessageList()
				}
			case StateMessageSelector:
				if len(m.SelectedMessages) > 0 {
					m.setupMonitoringTable()
					m.initializesTableDBCSignals()
					m.State = StateMonitoring
					go m.startReceavingMessages()
				} else {
					// Go back to send/receive selector if no messages selected
					m.State = StateSendReceiveSelector
				}
			case StateMonitoring:
				m.State = StateMessageSelector
				// Update the message list when returning from monitoring mode
				m.updateMessageListItems()
				// Reset the table to avoid it being visible
				m.MonitoringTable = table.Model{}
			case StateSending:
				// From sending mode, go back to send/receive selector
				m.State = StateSendReceiveSelector
				m.TextInput.SetValue("")
				m.LastSentMessage = ""
				m.SendStatus = ""
			}
		}

	case TickMsg:
		m.LastUpdate = time.Now()
		return m, TickCmd()
	}

	switch m.State {
	case StateFilePicker:
		m.FilePicker, cmd = m.FilePicker.Update(msg)
		cmds = append(cmds, cmd)

		// Check if a file was selected
		if didSelect, path := m.FilePicker.DidSelectFile(msg); didSelect {
			// Verify that it is a .dbc file
			if strings.HasSuffix(strings.ToLower(path), ".dbc") {
				m.DBCPath = path
				m.Err = m.loadDBC()
				if m.Err == nil {
					m.State = StateSendReceiveSelector
				}
			} else {
				m.Err = fmt.Errorf("select a .dbc file, not %s", path)
			}
		}

	case StateSendReceiveSelector:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.SendReceiveChoice > 0 {
					m.SendReceiveChoice--
				}
			case "down", "j":
				if m.SendReceiveChoice < 1 {
					m.SendReceiveChoice++
				}
			case "enter":
				if m.SendReceiveChoice == 0 {
					// Send mode
					m.State = StateSending
					m.TextInput.Focus()
				} else {
					// Receive mode
					m.State = StateMessageSelector
					m.setupMessageList()
				}
			}
		}

	case StateMessageSelector:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				if len(m.SelectedMessages) > 0 {
					m.setupMonitoringTable()
					m.initializesTableDBCSignals()
					m.State = StateMonitoring
					go m.startReceavingMessages()
				}
			case " ":
				// Toggle message selection
				m.toggleMessageSelection()
			}
		}
		m.MessageList, cmd = m.MessageList.Update(msg)
		cmds = append(cmds, cmd)

	case StateMonitoring:
		m.MonitoringTable, cmd = m.MonitoringTable.Update(msg)
		cmds = append(cmds, cmd)

	case StateSending:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// Send the message and clear the input
				if m.TextInput.Value() != "" {
					m.sendMessage(m.TextInput.Value())
					m.TextInput.SetValue("")
				}
			}
		}
		m.TextInput, cmd = m.TextInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
