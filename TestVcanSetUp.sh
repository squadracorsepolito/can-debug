#to install go run: sudo apt install golang-go
sudo apt update

#create virtual can network
sudo modprobe can
sudo modprobe can-raw
sudo modprobe vcan
sudo apt install can-utils #not neccessary but useful for testing (cansend, cangen, candump, ...)
sudo ip link add dev vcan0 type vcan
sudo ip link set up vcan0
#ip a show vcan0 ----> use to check if vcan0 has been succesfully created

#start program
go run main.go vcan0
