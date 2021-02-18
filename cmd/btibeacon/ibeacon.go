package main

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
    "fmt"
	"github.com/brutella/hc/log"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

type iBeacon struct {
	uuid  string
	major uint16
	minor uint16
}

func NewiBeacon(data []byte) (*iBeacon, error) {
	if len(data) < 25 || binary.BigEndian.Uint32(data) != 0x4c000215 {
		return nil, errors.New("Not an iBeacon")
	}
	beacon := new(iBeacon)
	beacon.uuid = strings.ToUpper(hex.EncodeToString(data[4:8]) + "-" + hex.EncodeToString(data[8:10]) + "-" + hex.EncodeToString(data[10:12]) + "-" + hex.EncodeToString(data[12:14]) + "-" + hex.EncodeToString(data[14:20]))
	beacon.major = binary.BigEndian.Uint16(data[20:22])
	beacon.minor = binary.BigEndian.Uint16(data[22:24])
	return beacon, nil
}

func onStateChanged(device gatt.Device, s gatt.State) {
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("Scanning for iBeacon Broadcasts...")
		device.Scan([]gatt.UUID{}, true)
		return
	default:
		device.StopScanning()
	}
}

func onPeripheralDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	b, err := NewiBeacon(a.ManufacturerData)
	if err == nil {
		fmt.Println("UUID: ", b.uuid)
		fmt.Println("Major: ", b.major)
		fmt.Println("Minor: ", b.minor)
		fmt.Println("RSSI: ", rssi)
	}
}

func main() {
	device, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Info.Printf("Failed to open device, err: %s\n", err)
		return
	}
	device.Handle(gatt.PeripheralDiscovered(onPeripheralDiscovered))
	device.Init(onStateChanged)
	select {}
}

