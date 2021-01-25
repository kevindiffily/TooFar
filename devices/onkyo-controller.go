package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

type OnkyoController struct {
	*accessory.Accessory

	MusicOptimizer *ToggleSvc
	Volume         *FaderSvc
	Dimmer         *FaderSvc
	Parent         interface{}
}

func NewOnkyoController(info accessory.Info) *OnkyoController {
	acc := OnkyoController{}
	acc.Accessory = accessory.New(info, accessory.TypeLightbulb)

	acc.MusicOptimizer = NewToggleSvc("Music Optimizer")
	acc.AddService(acc.MusicOptimizer.Service)

	acc.Dimmer = NewFaderSvc("Dimmer")
	acc.Dimmer.Value.SetMinValue(0)
	acc.Dimmer.Value.SetMaxValue(99)
	acc.Dimmer.Value.StepValue = 33
	acc.Dimmer.Value.SetValue(99)
	acc.AddService(acc.Dimmer.Service)

	acc.Volume = NewFaderSvc("Volume")
	acc.Volume.Value.SetMinValue(40)
	acc.Volume.Value.SetMaxValue(65)
	acc.Volume.Value.SetValue(55)
	acc.AddService(acc.Volume.Service)

	return &acc
}

type ToggleSvc struct {
	*service.Service

	On   *characteristic.On
	Name *characteristic.Name
}

func NewToggleSvc(name string) *ToggleSvc {
	svc := ToggleSvc{}
	svc.Service = service.New(service.TypeSwitch)

	svc.On = characteristic.NewOn()
	svc.AddCharacteristic(svc.On.Characteristic)

	svc.Name = characteristic.NewName()
	svc.Name.SetValue(name)
	svc.AddCharacteristic(svc.Name.Characteristic)

	return &svc
}

type FaderSvc struct {
	*service.Service

	Value  *characteristic.Brightness
	Name   *characteristic.Name
	Active *characteristic.Active
}

func NewFaderSvc(name string) *FaderSvc {
	svc := FaderSvc{}
	svc.Service = service.New(service.TypeLightbulb)

	svc.Value = characteristic.NewBrightness()
	svc.AddCharacteristic(svc.Value.Characteristic)

	svc.Name = characteristic.NewName()
	svc.Name.SetValue(name)
	svc.AddCharacteristic(svc.Name.Characteristic)

	svc.Active = characteristic.NewActive()
	svc.Active.SetValue(characteristic.ActiveActive)
	svc.AddCharacteristic(svc.Active.Characteristic)

	return &svc
}
