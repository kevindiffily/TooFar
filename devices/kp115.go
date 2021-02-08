package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

// KP115 is a single outlet with energy monitoring
type KP115 struct {
	*accessory.Accessory
	Outlet *KP115Svc
}

func NewKP115(info accessory.Info) *KP115 {
	acc := KP115{}
	acc.Accessory = accessory.New(info, accessory.TypeOutlet)

	acc.Outlet = NewKP115Svc()
	acc.AddService(acc.Outlet.Service)

	return &acc
}

type KP115Svc struct {
	*service.Service

	On          *characteristic.On
	OutletInUse *characteristic.OutletInUse

	ProgramMode       *characteristic.ProgramMode
	SetDuration       *characteristic.SetDuration
	RemainingDuration *characteristic.RemainingDuration
}

func NewKP115Svc() *KP115Svc {
	svc := KP115Svc{}
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
