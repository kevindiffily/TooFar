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
	Envoy       *envoy.Envoy
}

func NewEnvoy(info accessory.Info) *Envoy {
	acc := Envoy{}

	acc.Accessory = accessory.New(info, accessory.TypeSensor)
	acc.LightSensor = service.NewLightSensor()

	name := characteristic.NewName()
	name.SetValue("Solar Production")
	acc.LightSensor.AddCharacteristic(name.Characteristic)

	active := characteristic.NewActive()
	active.SetValue(characteristic.ActiveActive)
	acc.LightSensor.AddCharacteristic(active.Characteristic)

	acc.AddService(acc.LightSensor.Service)
	return &acc
}
