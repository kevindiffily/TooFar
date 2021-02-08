package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

// Both 200 and 210
type HS200 struct {
	*accessory.Accessory
	Switch *HS200Svc
}

func NewHS200(info accessory.Info) *HS200 {
	acc := HS200{}
	acc.Accessory = accessory.New(info, accessory.TypeSwitch)

	acc.Switch = NewHS200Svc()
	acc.AddService(acc.Switch.Service)
	return &acc
}

type HS200Svc struct {
	*service.Service

	On *characteristic.On

	ProgramMode       *characteristic.ProgramMode
	SetDuration       *characteristic.SetDuration
	RemainingDuration *characteristic.RemainingDuration
}

func NewHS200Svc() *HS200Svc {
	svc := HS200Svc{}
	svc.Service = service.New(service.TypeSwitch)

	svc.On = characteristic.NewOn()
	svc.AddCharacteristic(svc.On.Characteristic)

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
