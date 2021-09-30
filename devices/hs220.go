package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

type HS220 struct {
	*accessory.Accessory
	Lightbulb *HS220Svc
}

func NewHS220(info accessory.Info) *HS220 {
	acc := HS220{}
	acc.Accessory = accessory.New(info, accessory.TypeLightbulb)

	acc.Lightbulb = NewHS220Svc()
	acc.AddService(acc.Lightbulb.Service)
	return &acc
}

type HS220Svc struct {
	*service.Service

	On         *characteristic.On
	Brightness *characteristic.Brightness

	ProgramMode       *characteristic.ProgramMode
	SetDuration       *characteristic.SetDuration
	RemainingDuration *characteristic.RemainingDuration
}

func NewHS220Svc() *HS220Svc {
	svc := HS220Svc{}
	svc.Service = service.New(service.TypeLightbulb)

	svc.On = characteristic.NewOn()
	svc.AddCharacteristic(svc.On.Characteristic)

	svc.Brightness = characteristic.NewBrightness()
	svc.AddCharacteristic(svc.Brightness.Characteristic)

	svc.ProgramMode = characteristic.NewProgramMode()
	svc.AddCharacteristic(svc.ProgramMode.Characteristic)
	svc.ProgramMode.SetValue(characteristic.ProgramModeNoProgramScheduled)

	svc.SetDuration = characteristic.NewSetDuration()
	svc.AddCharacteristic(svc.SetDuration.Characteristic)

	svc.RemainingDuration = characteristic.NewRemainingDuration()
	svc.AddCharacteristic(svc.RemainingDuration.Characteristic)
	svc.RemainingDuration.SetValue(0)

	return &svc
}
