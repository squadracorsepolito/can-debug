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
		case StateSendConfiguration:
			m.SendTable.SetWidth(msg.Width)
			m.SendTable.SetHeight(msg.Height - 10)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			// Tab sempre torna indietro alla schermata precedente
			switch m.State {
			case StateSendReceiveSelector:
				// Da send/receive selector, torna a file picker solo se il DBC non è da riga di comando
				if !m.DBCFromCommandLine {
					m.State = StateFilePicker
				}
			case StateMessageSelector:
				// Da message selector, torna a send/receive selector
				m.State = StateSendReceiveSelector
			case StateMonitoring:
				// Da monitoring, torna a message selector
				m.State = StateMessageSelector
				// Update the message list when returning from monitoring mode
				m.updateMessageListItems()
				// Reset the table to avoid it being visible
				m.MonitoringTable = table.Model{}
			case StateSendConfiguration:
				// Da send configuration, torna a message selector
				m.State = StateMessageSelector
				// Update the message list when returning from send configuration
				m.updateMessageListItems()
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
					// Clear selected messages when changing mode
					m.SelectedMessages = []CANMessage{}
					// Only update if MessageList has been initialized
					if len(m.MessageList.Items()) > 0 {
						m.updateMessageListItems() // Update visual representation
					}
					// Note: don't update PreviousSendReceiveChoice here,
					// it will be updated when Enter is pressed
				}
			case "down", "j":
				if m.SendReceiveChoice < 1 {
					m.SendReceiveChoice++
					// Clear selected messages when changing mode
					m.SelectedMessages = []CANMessage{}
					// Only update if MessageList has been initialized
					if len(m.MessageList.Items()) > 0 {
						m.updateMessageListItems() // Update visual representation
					}
					// Note: don't update PreviousSendReceiveChoice here,
					// it will be updated when Enter is pressed
				}
			case "enter":
				// Check if mode actually changed since last time
				modeChanged := m.SendReceiveChoice != m.PreviousSendReceiveChoice

				if m.SendReceiveChoice == 0 {
					// Send mode - go to message selector
					m.State = StateMessageSelector
					if modeChanged {
						m.SelectedMessages = []CANMessage{} // Clear selection only if mode changed
					}
					m.setupMessageList()
				} else {
					// Receive mode
					m.State = StateMessageSelector
					if modeChanged {
						m.SelectedMessages = []CANMessage{} // Clear selection only if mode changed
					}
					m.setupMessageList()
				}

				// Update previous choice to current
				m.PreviousSendReceiveChoice = m.SendReceiveChoice
			}
		}

	case StateMessageSelector:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				if len(m.SelectedMessages) > 0 {
					if m.SendReceiveChoice == 0 {
						// Send mode - vai a send configuration
						m.State = StateSendConfiguration
						m.setupSendConfiguration()
					} else {
						// Receive mode - vai a monitoring
						m.setupMonitoringTable()
						m.initializesTableDBCSignals()
						m.State = StateMonitoring
						go m.startReceavingMessages()
					}
				}
			case " ":
				// Toggle message selection
				m.toggleMessageSelection()
			}
		}
		m.MessageList, cmd = m.MessageList.Update(msg)
		cmds = append(cmds, cmd)

	case StateMonitoring:
		// Update the table to handle scroll and cursor
		m.MonitoringTable, cmd = m.MonitoringTable.Update(msg)
		cmds = append(cmds, cmd)

		// Always ensure the table cursor is properly positioned and visible
		m.ensureTableCursorVisible()

	case StateSendConfiguration:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// Send once all signals of the current message
				m.sendSingleMessage()
			case " ":
				// Toggle start/stop for all signals of the current message
				_, ok := m.ActiveMessages[int(m.SelectedMessages[0].ID)]
				if ok {
					m.stopCyclicalSending()
				} else {
					m.startCyclicalSending()
				}
			case "right", "l":
				// Increase cycle time for all signals of the current message
				m.adjustMessageCycleTime(rangeMs)
			case "left":
				// Decrease cycle time for all signals of the current message
				m.adjustMessageCycleTime(-rangeMs)
			case "s":
				// Stop ALL message sending
				m.stopAllMessages()
			case "-":
				// Handle '-' for negative numbers in input field
				if m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
					// Check if we're in input mode and the input is focused
					currentInput := &m.SendSignals[m.CurrentInputIndex].TextInput
					if currentInput.Focused() {
						// Let the input handle the '-' for negative numbers
						*currentInput, cmd = currentInput.Update(msg)
						cmds = append(cmds, cmd)
						m.updateSendTableRows()
					}
				}
			case "up", "k":
				if m.CurrentInputIndex > 0 {
					// Remove focus from current input
					if m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
						m.SendSignals[m.CurrentInputIndex].TextInput.Blur()
					}
					// Move to previous input
					m.CurrentInputIndex--
					// Let the table handle the navigation, then sync
					m.SendTable, cmd = m.SendTable.Update(msg)
					cmds = append(cmds, cmd)
					// Sync table cursor with our index
					m.SendTable.SetCursor(m.CurrentInputIndex)
					// Set focus to new input
					if m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
						m.SendSignals[m.CurrentInputIndex].TextInput.Focus()
					}
				}
			case "down", "j":
				if m.CurrentInputIndex < len(m.SendSignals)-1 {
					// Remove focus from current input
					if m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
						m.SendSignals[m.CurrentInputIndex].TextInput.Blur()
					}
					// Move to next input
					m.CurrentInputIndex++
					// Let the table handle the navigation, then sync
					m.SendTable, cmd = m.SendTable.Update(msg)
					cmds = append(cmds, cmd)
					// Sync table cursor with our index
					m.SendTable.SetCursor(m.CurrentInputIndex)
					// Set focus to new input
					if m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
						m.SendSignals[m.CurrentInputIndex].TextInput.Focus()
					}
				}
			default:
				// Update the current input field
				if m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
					m.SendSignals[m.CurrentInputIndex].TextInput, cmd = m.SendSignals[m.CurrentInputIndex].TextInput.Update(msg)
					cmds = append(cmds, cmd)
					// Update the table rows to reflect the new input value
					m.updateSendTableRows()
				}
			}
		}

		// For non-navigation messages, let the table update normally to handle scroll and rendering
		if msg, ok := msg.(tea.KeyMsg); !ok || (msg.String() != "up" && msg.String() != "down" && msg.String() != "k" && msg.String() != "j") {
			m.SendTable, cmd = m.SendTable.Update(msg)
			cmds = append(cmds, cmd)
		}

		// Always ensure the table cursor is properly positioned and visible
		m.ensureTableCursorVisible()
	}

	return m, tea.Batch(cmds...)
}
