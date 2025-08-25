# CAN Debug Tool

A comprehensive CAN bus debugging tool with a modern Terminal User Interface (TUI) built with Bubble Tea. This tool provides advanced functionality for sending and receiving CAN messages with individual signal control and DBC file support.

## 🚀 Features

### Core Functionality

- **DBC File Support**: Load and parse DBC files for comprehensive CAN message definitions
- **Dual Mode Operation**: Choose between Send and Receive modes
- **Real-time Monitoring**: Live CAN message reception and signal decoding
- **Advanced Signal Transmission**: Individual frequency control for each signal
- **Modern TUI**: Beautiful terminal interface with table navigation and real-time updates

### Send Mode Features

- **Individual Signal Control**: Each signal has its own transmission frequency (50ms to 10s)
- **Flexible Sending Options**:
  - Single-shot transmission (Enter)
  - Continuous transmission with custom cycle times (Space)
  - Real-time cycle time adjustment (←→ arrows)
- **Emergency Stop**: Instantly stop all transmissions (s key)
- **Value Input Validation**: Support for decimal numbers, negative values, and floating-point precision
- **Live Status Indicators**: Visual feedback for active/inactive signals

### Receive Mode Features

- **Message Selection**: Choose specific CAN messages to monitor
- **Signal Decoding**: Automatic signal extraction and value interpretation using DBC definitions
- **Real-time Updates**: Live table updates with incoming CAN data
- **Search Functionality**: Filter messages by name for quick selection

## 📋 Requirements

### System Requirements

- **Linux**: Full SocketCAN support for sending and receiving
- **macOS**: Send support via can-utils (cansend command)
- **Windows**: Limited support (testing mode only)

### Dependencies

- Go 1.19 or higher
- Linux: SocketCAN interface (vcan0 or real CAN interface)
- macOS: can-utils package for sending

## 🛠️ Installation

### Build from Source

```bash
git clone <repository-url>
cd can-debug
go build -o can-debug
```

### Quick Start

```bash
# Using file picker
./can-debug

# Direct DBC file loading
./can-debug internal/test/MCB.dbc

# Show help
./can-debug -h
```

## 📖 Usage Guide

### Command Line Options

```bash
can-debug [file.dbc]     # Load DBC file directly
can-debug -h|--help      # Show comprehensive help
```

### Workflow

1. **File Selection**: Choose a DBC file using the built-in file picker or load directly
2. **Mode Selection**: Choose between Send or Receive mode
3. **Message Selection**: Select CAN messages for monitoring or configuration
4. **Operation**:
   - **Send Mode**: Configure signal values and transmission parameters
   - **Receive Mode**: Monitor live CAN traffic with signal decoding

### Keyboard Controls

#### Navigation Commands

| Key                  | Action                               |
| -------------------- | ------------------------------------ |
| `↑/↓` or `k/j` | Navigate up/down in lists and tables |
| `Tab`              | Go back to previous screen           |
| `q`                | Quit application                     |

#### File Selection

| Key       | Action                             |
| --------- | ---------------------------------- |
| `Enter` | Open directory or select .dbc file |

#### Send/Receive Mode Selection

| Key       | Action                              |
| --------- | ----------------------------------- |
| `↑/↓` | Choose between Send or Receive mode |
| `Enter` | Confirm selection                   |

#### Message List

| Key       | Action                        |
| --------- | ----------------------------- |
| `/`     | Search messages by name       |
| `Space` | Select/deselect message       |
| `Enter` | Confirm selection and proceed |

#### Send Configuration (Advanced Signal Control)

| Key       | Action                                |
| --------- | ------------------------------------- |
| `↑/↓` | Navigate between signals              |
| `Enter` | Send signal once (single shot)        |
| `Space` | Toggle continuous transmission        |
| `←/→` | Adjust cycle time (±50ms increments) |
| `s`     | Emergency stop all transmissions      |
| `Tab`   | Return to previous screen             |

**Signal Control Details:**

- **Cycle Time Range**: 50ms to 10,000ms (10 seconds)
- **Increment Size**: 50ms steps
- **Input Validation**: Supports integers, decimals, and negative values
- **Visual Indicators**: Active signals show ▶️, inactive show ⏸️
- **Single Shot Mode**: Temporary "-" indicator for one-time transmissions

#### Receive Mode (Monitoring)

| Key       | Action                          |
| --------- | ------------------------------- |
| `↑/↓` | Scroll through received signals |
| `Tab`   | Return to message selection     |

## 🏗️ Project Structure

```
can-debug/
├── main.go                    # Application entry point and CLI handling
├── internal/
│   ├── test/
│   │   └── MCB.dbc           # Example DBC file for testing
│   ├── ui/                   # User Interface package
│   │   ├── types.go          # Data structures and type definitions
│   │   ├── model.go          # Model initialization and state management
│   │   ├── update.go         # Event handling and state updates
│   │   ├── view.go           # UI rendering and display logic
│   │   └── handlers.go       # Business logic and CAN operations
│   └── can/
│       └── decoder.go        # CAN message decoding with acmelib
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
└── README.md                 # This file
```

### Key Components

#### `handlers.go` - Core Business Logic

- **DBC Management**: File loading, parsing, and message extraction
- **Signal Configuration**: Table setup and signal parameter management
- **Transmission Control**: Individual signal sending with task management
- **Monitoring Setup**: Real-time signal decoding and table updates

#### `update.go` - Event Processing

- **Keyboard Input**: Comprehensive key handling for all modes
- **State Transitions**: Smooth navigation between application states
- **Real-time Updates**: Window resizing and table cursor management

#### `view.go` - User Interface

- **Responsive Layout**: Dynamic sizing and content organization
- **Status Display**: Real-time feedback and instruction panels
- **Table Rendering**: Signal tables with proper alignment and formatting

#### `types.go` - Data Structures

- **Application State**: Model definitions and state management
- **Signal Representation**: Enhanced signal structures with transmission control
- **UI Components**: Table and input field configurations

## 🔧 Technical Details

### CAN Interface Support

#### Linux (Full Support)

```go
// SocketCAN connection
conn, err := socketcan.DialContext(context.Background(), "can", "vcan0")
```

#### macOS (Send Only)

```bash
# Uses cansend command for transmission
cansend vcan0 123#DEADBEEF
```

### DBC File Integration

The tool uses `acmelib` for comprehensive DBC file support:

```go
// DBC parsing and message extraction
database, err := acmelib.NewDatabaseFromFile(dbcPath)
messages := database.GetMessages()
```

### Signal Transmission Architecture

#### Task-Based Management

- **Concurrent Goroutines**: Each signal runs in its own goroutine
- **Unique Task IDs**: Proper cleanup and control of individual signals
- **Ticker-Based Timing**: Precise frequency control with time.Ticker

#### Example Implementation

```go
// Individual signal transmission
go func(signal *SendSignal, taskID int) {
    ticker := time.NewTicker(time.Duration(signal.CycleTime) * time.Millisecond)
    defer ticker.Stop()
  
    for {
        select {
        case <-ticker.C:
            // Send CAN message
            sendCANMessage(signal)
        case <-stopChannel:
            return
        }
    }
}(signal, taskID)
```

## 🧪 Testing

### Manual Testing with Example DBC

```bash
# Test with direct file loading
./can-debug internal/test/MCB.dbc

# Test file picker functionality
./can-debug
# Navigate to: internal/test/MCB.dbc
```

### Platform Testing

#### Linux with Virtual CAN

```bash
# Setup virtual CAN interface
sudo modprobe vcan
sudo ip link add dev vcan0 type vcan
sudo ip link set up vcan0

# Run tool with SocketCAN
./can-debug internal/test/MCB.dbc
```

#### macOS with can-utils

```bash
# Install can-utils (if available)
# Configure virtual interface
./can-debug internal/test/MCB.dbc
```

## 🤝 Contributing

We welcome contributions! Please follow these steps:

1. **Fork the Repository**
2. **Create a Feature Branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```
3. **Commit Your Changes**
   ```bash
   git commit -m 'Add amazing feature'
   ```
4. **Push to Branch**
   ```bash
   git push origin feature/amazing-feature
   ```
5. **Open a Pull Request**

### Development Guidelines

- Follow Go conventions and best practices
- Maintain TUI responsiveness and user experience
- Add comprehensive comments for complex logic
- Test on multiple platforms when possible

## 📚 Example Usage Scenarios

### Automotive Testing

```bash
# Load vehicle DBC file
./can-debug vehicle_network.dbc

# Select powertrain messages
# Configure RPM signal: 100ms cycle time
# Configure throttle position: 50ms cycle time
# Start continuous transmission for engine simulation
```

### Industrial Automation

```bash
# Load machine control DBC
./can-debug machine_control.dbc

# Monitor sensor feedback signals
# Send actuator commands with precise timing
# Use emergency stop for safety testing
```

### Development and Debugging

```bash
# Quick signal testing
./can-debug test_network.dbc

# Single-shot signal verification
# Cycle time optimization testing
# Protocol compliance validation
```

## 🐛 Troubleshooting

### Common Issues

#### "No CAN interface found"

- **Linux**: Ensure SocketCAN interface is up (`ip link show`)
- **macOS**: Install can-utils package
- **Solution**: Tool continues in test mode if no interface available

#### "DBC file parsing error"

- **Cause**: Invalid or corrupted DBC file
- **Solution**: Verify DBC file format and syntax

#### "Signal transmission not working"

- **Linux**: Check SocketCAN permissions and interface status
- **macOS**: Verify can-utils installation and virtual interface setup
