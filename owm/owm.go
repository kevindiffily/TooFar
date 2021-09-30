package owm

import (
	owm "github.com/briandowns/openweathermap"

	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/util"
	"strconv"
	"sync"
	"time"
)

// Platform is the handle to the OWM sensors
type Platform struct {
	Running bool
}

var owms map[string]*tfaccessory.TFAccessory
var doOnce sync.Once

// Startup is called by the platform management to get things going
func (o Platform) Startup(c *config.Config) platform.Control {
	o.Running = true
	return o
}

// Shutdown is called by the platform management to shut things down
func (o Platform) Shutdown() platform.Control {
	o.Running = false
	return o
}

// AddAccessory adds an OWM location and registers it with HC
func (o Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	doOnce.Do(func() {
		owms = make(map[string]*tfaccessory.TFAccessory)
	})

	if a.Info.Name == "" {
		a.Info.Name = a.Username
	}
	if a.Info.Manufacturer == "" {
		a.Info.Manufacturer = "TooFar"
	}
	a.Info.Model = "OpenWeatherMap"

	storage, err := util.NewFileStorage("serials")
	if err != nil {
		log.Info.Println("unable to get storage")
	}
	serial := util.GetSerialNumberForAccessoryName(a.Info.Name, storage)
	a.Info.SerialNumber = serial

	// if an ID number isn't specified, use the serial number (consistent across restarts) to generate one
	if a.Info.ID == 0 {
		i, err := strconv.ParseUint(serial[0:8], 16, 64)
		if err != nil {
			log.Info.Println(err.Error())
		}
		a.Info.ID = i
	}

	a.Type = accessory.TypeSensor

	owms[a.Name] = a

	a.Device = devices.NewOpenWeatherMap(a.Info)
	a.Accessory = a.Device.(*devices.OpenWeatherMap).Accessory

	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)
	a.UpdateIDs()

	w, err := owm.NewCurrent("C", "EN", a.Password)
	if err != nil {
		log.Info.Println(err.Error())
		return
	}
	w.CurrentByName(a.Username)
	log.Info.Printf("%+v", w.Main)
	owmdev := a.Device.(*devices.OpenWeatherMap)
	if owmdev.TemperatureSensor.CurrentTemperature.GetValue() != w.Main.Temp {
		owmdev.TemperatureSensor.CurrentTemperature.SetValue(w.Main.Temp)
	}

	if owmdev.HumiditySensor.CurrentRelativeHumidity.GetValue() != float64(w.Main.Humidity) {
		owmdev.HumiditySensor.CurrentRelativeHumidity.SetValue(float64(w.Main.Humidity))
	}
}

// GetAccessory looks up an OWM sensor
func (o Platform) GetAccessory(name string) (*tfaccessory.TFAccessory, bool) {
	val, ok := owms[name]
	return val, ok
}

// Background starts up the go process to periodically update the sensors values
func (o Platform) Background() {
	go func() {
		for range time.Tick(time.Minute * 5) {
			o.backgroundPuller()
		}
	}()
}

func (o Platform) backgroundPuller() {
	for _, a := range owms {
		w, err := owm.NewCurrent("C", "EN", a.Password)
		if err != nil {
			log.Info.Println(err.Error())
		}
		w.CurrentByName(a.Username)
		owmdev := a.Device.(*devices.OpenWeatherMap)
		if owmdev.TemperatureSensor.CurrentTemperature.GetValue() != w.Main.Temp {
			owmdev.TemperatureSensor.CurrentTemperature.SetValue(w.Main.Temp)
		}
		if owmdev.HumiditySensor.CurrentRelativeHumidity.GetValue() != float64(w.Main.Humidity) {
			owmdev.HumiditySensor.CurrentRelativeHumidity.SetValue(float64(w.Main.Humidity))
		}
	}
}
