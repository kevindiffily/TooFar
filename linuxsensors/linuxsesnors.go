package linuxsensors

import (
	"github.com/ssimunic/gosensors"

	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/platform"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	// "github.com/brutella/hc/service"
	"strconv"
	"time"
)

var sensors *tfaccessory.TFAccessory

// Platform is the handle to the sensors
type Platform struct {
	Running bool
}

// Startup is called by the platform management to get things going
func (s Platform) Startup(c config.Config) platform.Control {
	s.Running = true
	return s
}

// Shutdown is called by the platform management to shut things down
func (s Platform) Shutdown() platform.Control {
	s.Running = false
	return s
}

// AddAccessory adds the host senors HC
func (s Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	a.Platform = "LinuxSensors"
	a.Name = "OS Sensors"
	a.Type = accessory.TypeSensor
	a.Info.Name = "OS Sensors"
	a.Info.Manufacturer = "Linux"
	a.Info.ID = 102
	a.Info.SerialNumber = "0"
	a.Info.FirmwareRevision = "0.0.0"
	a.Runner = actionRunner

	nfs, err := gosensors.NewFromSystem()
	if err != nil {
		log.Info.Println(err)
		return
	}

	// add to HC for GUI
	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)
	if a.Thermometer == nil || a.Thermometer.TempSensor == nil {
		log.Info.Println("unable to create sensor type")
		return
	}
	a.Thermometer.TempSensor.CurrentTemperature.Description = "CurrentTemperature"
	a.Thermometer.TempSensor.Primary = true

	for chip := range nfs.Chips {
		for k, v := range nfs.Chips[chip] {
			if k == "temp1" {
				temp, err := strconv.ParseFloat(v[1:5], 64)
				if err != nil {
					log.Info.Println(err)
				} else {
					log.Info.Printf("setting OS temp to: %f", temp)
					a.Thermometer.TempSensor.CurrentTemperature.SetValue(temp)
				}
			}
			log.Info.Println(k, v)
		}
	}

	sensors = a
}

func actionRunner(a *tfaccessory.TFAccessory, d *action.Action) {
	log.Info.Printf("in linuxsensors action runner: %+v %+v", a, d)
}

// GetAccessory looks up an sensor
func (s Platform) GetAccessory(name string) (*tfaccessory.TFAccessory, bool) {
	return sensors, true
}

// Background starts up the go process to periodically update the sensors values
func (s Platform) Background() {
	go func() {
		for range time.Tick(time.Minute * 5) {
			s.backgroundPuller()
		}
	}()
}

func (s Platform) backgroundPuller() {
	nfs, err := gosensors.NewFromSystem()
	if err != nil {
		log.Info.Println(err)
		return
	}
	for chip := range nfs.Chips {
		for k, v := range nfs.Chips[chip] {
			if k == "temp1" {
				sensors.Thermometer.TempSensor.CurrentTemperature.Description = "CurrentTemperature"
				temp, _ := strconv.ParseFloat(v[1:5], 64)
				if temp != sensors.Thermometer.TempSensor.CurrentTemperature.GetValue() {
					// log.Info.Printf("setting OS temp to: %f", temp)
					sensors.Thermometer.TempSensor.CurrentTemperature.SetValue(temp)
				}
			}
		}
	}
}
