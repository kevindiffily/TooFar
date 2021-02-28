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
	LightSensor *service.LightSensor
	Active      *characteristic.Active
	Envoy       *envoy.Envoy
}

func NewEnvoy(info accessory.Info) *Envoy {
	acc := Envoy{}

	acc.Accessory = accessory.New(info, accessory.TypeSensor)
	acc.LightSensor = service.NewLightSensor()

	name := characteristic.NewName()
	name.SetValue("Solar Production")
	acc.LightSensor.AddCharacteristic(name.Characteristic)

	acc.Active = characteristic.NewActive()
	acc.Active.SetValue(characteristic.ActiveActive)
	acc.LightSensor.AddCharacteristic(acc.Active.Characteristic)

	acc.AddService(acc.LightSensor.Service)
	return &acc
}
