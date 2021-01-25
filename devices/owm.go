package devices

import (
	"fmt"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

// a system might have several chips
type OpenWeatherMap struct {
	*accessory.Accessory

	TemperatureSensor *service.TemperatureSensor
	HumiditySensor    *service.HumiditySensor
	AirQualitySensor  *service.AirQualitySensor
}

func NewOpenWeatherMap(info accessory.Info) *OpenWeatherMap {
	acc := OpenWeatherMap{}
	acc.Accessory = accessory.New(info, accessory.TypeSensor)

	acc.TemperatureSensor = service.NewTemperatureSensor()
	acc.Accessory.AddService(acc.TemperatureSensor.Service)
	acc.TemperatureSensor.CurrentTemperature.Description = fmt.Sprintf("%s Temp", info.Name)

	acc.HumiditySensor = service.NewHumiditySensor()
	acc.Accessory.AddService(acc.HumiditySensor.Service)
	acc.HumiditySensor.CurrentRelativeHumidity.Description = fmt.Sprintf("%s Humidity", info.Name)

	// does not seem to work
	acc.AirQualitySensor = service.NewAirQualitySensor()
	acc.AirQualitySensor.AirQuality.SetValue(characteristic.AirQualityGood)
	// acc.Accessory.AddService(acc.AirQualitySensor.Service)
	acc.AirQualitySensor.AirQuality.Description = fmt.Sprintf("%s AQ", info.Name)

	return &acc
}
