package linuxsensors

import (
	"github.com/ssimunic/gosensors"

	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"

	"fmt"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/service"
	"github.com/brutella/hc/util"
	"strconv"
	"time"
)

var sensors *tfaccessory.TFAccessory

// Platform is the handle to the sensors
type Platform struct {
	Running bool
}

// Startup is called by the platform management to get things going
func (s Platform) Startup(c *config.Config) platform.Control {
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
	storage, err := util.NewFileStorage("serials")
	if err != nil {
		log.Info.Println("unable to get storage")
	}
	serial := util.GetSerialNumberForAccessoryName("LinuxSensors", storage)

	a.Platform = "LinuxSensors"
	a.Name = "OS Sensors"
	a.Type = accessory.TypeSensor
	a.Info.Name = "OS Sensors"
	a.Info.Model = "OS Sensors"
	a.Info.Manufacturer = "Linux"
	a.Info.ID = 103
	a.Info.SerialNumber = serial
	a.Info.FirmwareRevision = "0.0.3"
	a.Runner = actionRunner

	nfs, err := gosensors.NewFromSystem()
	if err != nil {
		log.Info.Println(err)
		return
	}

	// add to HC for GUI
	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)
	if a.LinuxSensors == nil {
		log.Info.Println("unable to create LinuxSensors type")
		return
	}

	noprimary := true
	for chip := range nfs.Chips {
		scv := make(devices.SensorChipValues)
		a.LinuxSensors.Chips[chip] = &scv
		for k, v := range nfs.Chips[chip] {
			if k == "temp1" { // change this to a switch, handle fans and other temps as well
				scv[k] = service.NewTemperatureSensor()
				name := characteristic.NewName()
				scv[k].AddCharacteristic(name.Characteristic)
				name.SetValue(fmt.Sprintf("%s/%s", chip, k))
				a.LinuxSensors.AddService(scv[k].Service)
				if noprimary {
					scv[k].Primary = true
					noprimary = false
				}
				temp, err := strconv.ParseFloat(v[1:5], 64)
				if err != nil {
					log.Info.Println(err)
				} else {
					// log.Info.Printf("setting %s/%s temp to: %f", chip, k, temp)
					scv[k].CurrentTemperature.SetValue(temp)
				}
			}
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
	a, _ := s.GetAccessory("OS Sensors")
	for chip := range nfs.Chips {
		for k, v := range nfs.Chips[chip] {
			if k == "temp1" { // switch, handle various types...
				temp, err := strconv.ParseFloat(v[1:5], 64)
				if err != nil {
					log.Info.Println(err)
				} else {
					// log.Info.Printf("setting %s/%s temp to: %f", chip, k, temp)
					s, ok := a.LinuxSensors.Chips[chip]
					if ok {
						(*s)[k].CurrentTemperature.SetValue(temp)
					}
				}
			}
		}
	}
}
