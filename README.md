# CAN Debug Tool

A CAN bus debugging tool with a Terminal User Interface (TUI) built with Bubble Tea. This tool provides advanced functionality for sending and receiving CAN messages with individual message control and DBC file support.

## 🚀 Features

### Core Functionality

- **DBC File Support**: Load and parse DBC files for comprehensive CAN message definitions
- **Dual Mode Operation**: Choose between Send and Receive modes
- **Real-time Monitoring**: Live CAN message reception and signal decoding
- **Signal Transmission**: Individual frequency control for each message

### Send Mode Features

- **Individual Message Control**: Each message has its own transmission frequency (10ms to 10s) *(can be modified by changing the value rangeMs in internal/ui/types.go)*
- **Sending Options**:
  - Single-shot transmission
  - Continuous transmission with custom cycle times
- **Emergency Stop**: Instantly stop all transmissions

### Receive Mode Features

- **Message Selection**: Choose specific CAN messages to monitor
- **Signal Decoding**: Automatic signal extraction and value interpretation using DBC definitions

## 📋 Requirements

- SocketCAN interface (vcan0 or real CAN interface)  --->  **you need to have linux or some emulator like WSL**

## 🛠️ Installation and Usage Guide

### Quick Installation (Recommended)

Download the latest pre-built executable from the [Releases page](https://github.com/squadracorsepolito/can-debug/releases):

1. **For Linux (x86_64)**: Download `can-debug_Linux_x86_64.tar.gz`
2. **For macOS (Intel)**: Download `can-debug_Darwin_x86_64.tar.gz`
3. **For macOS (Apple Silicon)**: Download `can-debug_Darwin_arm64.tar.gz`
4. **For Windows**: Download `can-debug_Windows_x86_64.zip`

Extract and run:

```bash
# Linux/macOS
tar -xzf can-debug_*.tar.gz
./can-debug

# Windows (extract zip and run)
can-debug.exe
```

### Build from Source (For Development)

Requirements:

- Go 1.19 or higher

```bash
# Install Go on Ubuntu/Debian
sudo apt install golang-go
```

Build steps:

```bash
git clone https://github.com/squadracorsepolito/can-debug.git
cd can-debug
go build -o can-debug
```

### Command Line Options

⚠️⚠️⚠️ **Before running you should set up a can or vcan network** ️⚠️⚠️️⚠️

```bash
can-debug [canNetworkName]            # Use the file picker to choose the dbc file
can-debug [canNetworkName] [file.dbc] # Load DBC file directly
can-debug -h|--help                   # Show comprehensive help
```

## 🧪 Testing

The application can be tested using a virtual CAN network (vcan) or a real CAN interface. Two approaches are provided: a quick helper script and a manual setup.

### With vcan (quick)

1. Run the helper script to create a vcan interface:

```bash
./TestVcanSetUp.sh
```

2. Start the application in one terminal (example):

```bash
./can-debug vcan0
```

3. In another terminal verify behaviour:

- To observe frames sent by the program:

```bash
candump vcan0
```

- To send a frame to the program (example):

```bash
cansend vcan0 123#DEADBEEF
```

Notes:

- Use `internal/test/MCB.dbc` as a sample DBC file for testing.
- Press `Ctrl+C` to stop `candump`.

### With vcan (manual)

If you prefer to create the virtual CAN interface manually:

```bash
# create virtual CAN network
sudo modprobe can
sudo modprobe can-raw
sudo modprobe vcan
sudo apt install can-utils    # optional but useful (cansend, cangen, candump)
sudo ip link add dev vcan0 type vcan
sudo ip link set up vcan0
# verify:
ip a show vcan0
```

### With a real CAN interface

Replace `[CanNetworkName]` and `[bitrate]` with your interface name and bitrate:

```bash
sudo modprobe can
sudo modprobe can-raw
sudo modprobe <appropriate_module>
sudo apt install can-utils    # optional but useful
sudo ip link set <CanNetworkName> type can bitrate <bitrate>
sudo ip link set up <CanNetworkName>
ip a show <CanNetworkName>
```

After the interface is up, run the program specifying the interface and optionally a DBC file:

```bash
# use the file picker:
./can-debug <CanNetworkName>

# or specify DBC directly:
./can-debug <CanNetworkName> path/to/file.dbc
```

Common test commands (same as above):

```bash
candump <interface>
cansend <interface> 123#DEADBEEF
```

## 🤝 Contributing

### Development Setup

For contributors who want to modify the code:

1. **Fork and clone** the repository
2. **Install Go 1.19+** if not already installed
3. **Build from source** using the instructions above
4. **Test your changes** using the vcan setup
5. **Submit a Pull Request** to the `dev` branch
