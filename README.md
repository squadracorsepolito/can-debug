# CAN Debug Tool

A comprehensive CAN bus debugging tool with a Terminal User Interface (TUI) built with Bubble Tea. This tool provides advanced functionality for sending and receiving CAN messages with individual message control and DBC file support.

## 🚀 Features

### Core Functionality

- **DBC File Support**: Load and parse DBC files for comprehensive CAN message definitions
- **Dual Mode Operation**: Choose between Send and Receive modes
- **Real-time Monitoring**: Live CAN message reception and signal decoding
- **Advanced Signal Transmission**: Individual frequency control for each message

### Send Mode Features

- **Individual Message Control**: Each message has its own transmission frequency (10ms to 10s) ***(can be changed by changing the value rangeMs in internal/ui/types.go)***
- **Sending Options**:
  - Single-shot transmission (Enter)
  - Continuous transmission with custom cycle times (Space)
- **Emergency Stop**: Instantly stop all transmissions (s key)

### Receive Mode Features

- **Message Selection**: Choose specific CAN messages to monitor
- **Signal Decoding**: Automatic signal extraction and value interpretation using DBC definitions

## 📋 Requirements

- Go 1.19 or higher. 
You can install it with: 
```bash
#installing go
sudo apt install golang-go
```

- SocketCAN interface (vcan0 or real CAN interface)  --->  **you need to have linux or some emulator like WSL**

## 🛠️ Installation and Usage Guide

### Build from Source
```bash
git clone <repository-url>
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

### With Vcan Network
#### 1)If you want to quickly test the progragram, you just need to run the file 'TestVcanSetUp' like this
```bash
./TestVcanSetUp.sh
```
#### in order to test if the program is working correclty:
- run the program in a window
- open another window and run the following commands to test
    - Test for sending
        - run this command: 'candump vcan0'
        - now every time the main program sends a message you should see it on the candump window 
        - press ctrl+c to stop the candump
    - Test for receiving
        - go in the monitoring mode on the main program
        - on the other window use 'cansend vcan0 [Message ID]#[Data] (example: cansend vcan0 123#DEADBEEF)
        - if you are monitoring the message with that same ID you shoud receive it on the main program


You can use the file present in 'internal/test/MCB.dbc' in order to test

#### 2) If you want to do it manually:

```bash
#create virtual can network
sudo modprobe can
sudo modprobe can-raw
sudo modprobe vcan
sudo apt install can-utils #not neccessary but useful for testing (cansend, cangen, candump, ...)
sudo ip link add dev [VCanNetworkName] type vcan
sudo ip link set up [VCanNetworkName]
#ip a show vcan0 ----> use to check if vcan0 has been succesfully created
```

### With Real Can Network:

```bash
#create virtual can network
sudo modprobe can
sudo modprobe can-raw
sudo modprobe [AppropriateModuleName]
sudo apt install can-utils #not neccessary but useful for testing (cansend, cangen, candump, ...)
sudo ip link set [CanNetworkName] type can bitrate [bitrate]
sudo ip link set up [CanNetworkName]
#ip a show vcan0 ----> use to check if vcan0 has been succesfully created
```
