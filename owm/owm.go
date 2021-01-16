package owm

import (
	owm "github.com/briandowns/openweathermap"

	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/platform"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/service"
	"github.com/brutella/hc/util"
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
		a.Info.Manufacturer = "OpenWeatherMap"
	}
	if a.Info.ID == 0 {
		a.Info.ID = 1341
	}

	storage, err := util.NewFileStorage("serials")
	if err != nil {
		log.Info.Println("unable to get storage")
	}
	serial := util.GetSerialNumberForAccessoryName(a.Info.Name, storage)
	a.Info.SerialNumber = serial

	a.Type = accessory.TypeSensor

	owms[a.Name] = a

	// add to HC for GUI
	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)

	a.Runner = actionRunner

	w, err := owm.NewCurrent("C", "EN", a.Password)
	if err != nil {
		log.Info.Println(err.Error())
		// a.StatusActive.SetValue(false)
		return
	}
	w.CurrentByName(a.Username)
	log.Info.Printf("%+v", w.Main)
	a.Thermometer.TempSensor.CurrentTemperature.Description = "CurrentTemperature"
	if a.Thermometer.TempSensor.CurrentTemperature.GetValue() != w.Main.Temp {
		a.Thermometer.TempSensor.CurrentTemperature.SetValue(w.Main.Temp)
	}

	a.HumiditySensor = service.NewHumiditySensor()
	a.Accessory.AddService(a.HumiditySensor.Service)
	a.HumiditySensor.CurrentRelativeHumidity.Description = "CurrentRelativeHumidity"
	if a.HumiditySensor.CurrentRelativeHumidity.GetValue() != float64(w.Main.Humidity) {
		a.HumiditySensor.CurrentRelativeHumidity.SetValue(float64(w.Main.Humidity))
	}
}

// actionRunner here makes no sense...
func actionRunner(a *tfaccessory.TFAccessory, d *action.Action) {
	log.Info.Printf("in owm action runner: %+v %+v", a, d)
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
		// log.Info.Printf("%+v", w.Main)
		// if w.Main.Temp == 0 && w.Main.Humidity == 0 { a.StatusActive.SetValue(false) } else if a.StatusActive.GetValue() == false { a.StatusActive.SetValue(true) }
		if a.Thermometer != nil && a.Thermometer.TempSensor.CurrentTemperature.GetValue() != w.Main.Temp {
			a.Thermometer.TempSensor.CurrentTemperature.SetValue(w.Main.Temp)
		}
		if a.HumiditySensor != nil && a.HumiditySensor.CurrentRelativeHumidity.GetValue() != float64(w.Main.Humidity) {
			a.HumiditySensor.CurrentRelativeHumidity.SetValue(float64(w.Main.Humidity))
		}
	}
}
