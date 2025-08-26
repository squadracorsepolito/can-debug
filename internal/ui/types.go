package ui

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/squadracorsepolito/acmelib"
	"go.einride.tech/can/pkg/socketcan"

	"github.com/squadracorsepolito/can-debug/internal/can"
)

// State represents the current state of the UI
type State int

const rangeMs int = 10 

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

// infoSending contains info of the message being currenty send (cyclically)
// frequancy is the frequency at wich is being sent (in ms)
// stop is the function that needs to be call in order to stop the sending
type infoSending struct{
	frequency int
	stop context.CancelFunc
}

// SendSignal represents a signal to be sent with its input field
type SendSignal struct {
	SignalName   string
	Unit         string
	TextInput    textinput.Model
	Value        string
	IsActive     bool // whether this signal/message is actively being sent
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
	Transmitter        socketcan.Transmitter
 	// send/receive functionality
	SendReceiveChoice         int // 0 = send, 1 = receive
	PreviousSendReceiveChoice int // to track when mode actually changes
	SendStatus                string // Status message for sending operations
	// send configuration fields
	SendSignals       []SendSignal
	SendTable         table.Model
	CurrentInputIndex int // which input is currently focused
	CycleTime         int
	// data structure for message sending

	ActiveMessages map[int]infoSending //map of messageID -> struct with info of the message being currenty send (cyclically) 





	
}

// Message for updating real-time data
type TickMsg time.Time
