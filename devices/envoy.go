package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
	"github.com/cloudkucooland/go-envoy"
)

// present the solar production as Lux sensor
type Envoy struct {
	*accessory.Accessory
	LightSensor     *service.LightSensor
	Active          *characteristic.Active
	DailyProduction *service.BatteryService
	Envoy           *envoy.Envoy
}

func NewEnvoy(info accessory.Info) *Envoy {
	acc := Envoy{}

	// use a light-sensor type for the current production
	acc.Accessory = accessory.New(info, accessory.TypeSensor)
	acc.LightSensor = service.NewLightSensor()

	name := characteristic.NewName()
	name.SetValue("Solar Production")
	acc.LightSensor.AddCharacteristic(name.Characteristic)

	acc.Active = characteristic.NewActive()
	acc.Active.SetValue(characteristic.ActiveActive)
	acc.LightSensor.AddCharacteristic(acc.Active.Characteristic)

	acc.AddService(acc.LightSensor.Service)

	// use a battery type to expose the daily total
	acc.DailyProduction = service.NewBatteryService()
	acc.DailyProduction.BatteryLevel.SetValue(0)

	blname := characteristic.NewName()
	blname.SetValue("Daily Solar Total")
	acc.DailyProduction.AddCharacteristic(blname.Characteristic)

	acc.AddService(acc.DailyProduction.Service)
	return &acc
}
