package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

// HS103 is a single outlet without energy monitoring
type HS103 struct {
	*accessory.Accessory
	Outlet *HS103Svc
}

func NewHS103(info accessory.Info) *HS103 {
	acc := HS103{}
	acc.Accessory = accessory.New(info, accessory.TypeOutlet)

	acc.Outlet = NewHS103Svc()
	acc.AddService(acc.Outlet.Service)

	return &acc
}

type HS103Svc struct {
	*service.Service

	On          *characteristic.On
	OutletInUse *characteristic.OutletInUse

	ProgramMode       *characteristic.ProgramMode
	SetDuration       *characteristic.SetDuration
	RemainingDuration *characteristic.RemainingDuration
}

func NewHS103Svc() *HS103Svc {
	svc := HS103Svc{}
	svc.Service = service.New(service.TypeLightbulb)

	svc.On = characteristic.NewOn()
	svc.AddCharacteristic(svc.On.Characteristic)

	svc.OutletInUse = characteristic.NewOutletInUse()
	svc.AddCharacteristic(svc.OutletInUse.Characteristic)

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
