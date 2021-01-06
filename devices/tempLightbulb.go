package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

type TempLightbulb struct {
	*accessory.Accessory
	Lightbulb *TempLightbulbSvc
}

func NewTempLightbulb(info accessory.Info) *TempLightbulb {
	acc := TempLightbulb{}
	acc.Accessory = accessory.New(info, accessory.TypeLightbulb)
	acc.Lightbulb = NewTempLightbulbSvc()

	acc.AddService(acc.Lightbulb.Service)

	return &acc
}

type TempLightbulbSvc struct {
	*service.Service

	On               *characteristic.On
	Brightness       *characteristic.Brightness
	ColorTemperature *characteristic.ColorTemperature
}

func NewTempLightbulbSvc() *TempLightbulbSvc {
	svc := TempLightbulbSvc{}
	svc.Service = service.New(service.TypeLightbulb)

	svc.On = characteristic.NewOn()
	svc.AddCharacteristic(svc.On.Characteristic)

	svc.Brightness = characteristic.NewBrightness()
	svc.AddCharacteristic(svc.Brightness.Characteristic)

	svc.ColorTemperature = characteristic.NewColorTemperature()
	svc.AddCharacteristic(svc.ColorTemperature.Characteristic)

	return &svc
}
