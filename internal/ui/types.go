package ui

import (
	"fmt"
	"net"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/squadracorsepolito/acmelib"

	"github.com/squadracorsepolito/can-debug/internal/can"
)

// State represents the current state of the UI
type State int

const (
	StateFilePicker State = iota
	StateSendReceiveSelector
	StateMessageSelector
	StateMonitoring
	StateSendConfiguration
)

// CANMessage represents a message in the CAN bus
type CANMessage struct {
	ID       uint32
	Name     string
	Selected bool
	Message  *acmelib.Message
}

func (c CANMessage) Title() string {
	if c.Selected {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AA00FF")).
			Bold(true).
			Render(c.Name)
	}
	return c.Name
}

func (c CANMessage) Description() string {
	desc := fmt.Sprintf("ID: 0x%X", c.ID)
	if c.Selected {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8800CC")).
			Render(desc)
	}
	return desc
}

func (c CANMessage) FilterValue() string { return c.Name }

// SendSignal represents a signal to be sent with its input field
type SendSignal struct {
	MessageName  string
	ID           uint32
	SignalName   string
	Unit         string
	TextInput    textinput.Model
	Value        string
	CycleTime    int  // cycle time in milliseconds for this specific message
	IsActive     bool // whether this signal/message is actively being sent
	TaskID       int  // unique ID for the task (for stopping)
	IsSingleShot bool // true if this is a single shot send (shows "-" in cycle column)
}

// Main model of the application
type Model struct {
	State              State
	FilePicker         filepicker.Model
	MessageList        list.Model
	MonitoringTable    table.Model
	SelectedMessages   []CANMessage
	DBCPath            string
	DBCFromCommandLine bool // true if DBC file was provided via command line
	Messages           []*acmelib.Message
	Decoder            *can.Decoder
	LastUpdate         time.Time
	Width              int
	Height             int
	Err                error
	CanNetwork         net.Conn
	// send/receive functionality
	SendReceiveChoice         int // 0 = send, 1 = receive
	PreviousSendReceiveChoice int // to track when mode actually changes
	TextInput                 textinput.Model
	LastSentMessage           string
	SendStatus                string // Status message for sending operations
	// send configuration fields
	SendSignals       []SendSignal
	SendTable         table.Model
	CurrentInputIndex int // which input is currently focused
	// task management for individual message sending
	NextTaskID  int                   // counter for unique task IDs
	ActiveTasks map[int]chan struct{} // map of taskID -> stop channel
	// cyclical sending options (global, deprecated - use per-message cycle time)
	SendMode          int  // 0 = single send, 1 = cyclical send
	SendInterval      int  // interval in milliseconds for cyclical sending (default 100ms)
	IsSendingCyclical bool // flag to track if cyclical sending is active
}

// Message for updating real-time data
type TickMsg time.Time
