package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

type HS220 struct {
	*accessory.Accessory
	Lightbulb          *HS220Svc
	ProgrammableSwitch *service.StatefulProgrammableSwitch
	WifiTransport      *service.WifiTransport
}

func NewHS220(info accessory.Info) *HS220 {
	acc := HS220{}
	acc.Accessory = accessory.New(info, accessory.TypeLightbulb)

	acc.Lightbulb = NewHS220Svc()
	acc.AddService(acc.Lightbulb.Service)

	acc.ProgrammableSwitch = service.NewStatefulProgrammableSwitch()
	acc.AddService(acc.ProgrammableSwitch.Service)

	acc.ProgrammableSwitch.ProgrammableSwitchEvent = characteristic.NewProgrammableSwitchEvent()
	acc.ProgrammableSwitch.ProgrammableSwitchEvent.SetValue(0)
	acc.ProgrammableSwitch.AddCharacteristic(acc.ProgrammableSwitch.ProgrammableSwitchEvent.Characteristic)

	acc.ProgrammableSwitch.ProgrammableSwitchOutputState = characteristic.NewProgrammableSwitchOutputState()
	acc.ProgrammableSwitch.AddCharacteristic(acc.ProgrammableSwitch.ProgrammableSwitchOutputState.Characteristic)

	acc.WifiTransport = service.NewWifiTransport()
	acc.WifiTransport.CurrentTransport.SetValue(true)
	// acc.AddService(acc.WifiTransport.Service)

	return &acc
}

type HS220Svc struct {
	*service.Service

	On         *characteristic.On
	Brightness *characteristic.Brightness
}

func NewHS220Svc() *HS220Svc {
	svc := HS220Svc{}
	svc.Service = service.New(service.TypeLightbulb)

	svc.On = characteristic.NewOn()
	svc.AddCharacteristic(svc.On.Characteristic)

	svc.Brightness = characteristic.NewBrightness()
	svc.AddCharacteristic(svc.Brightness.Characteristic)

	return &svc
}
