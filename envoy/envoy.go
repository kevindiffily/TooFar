package envoy

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"
	// "net/http"
	"sync"
	"time"
	// "io/ioutil"
	"github.com/cloudkucooland/go-envoy"
	"strconv"
)

// Platform is the platform handle for the Kasa stuff
type Platform struct {
	Running bool
}

var envoys map[string]*tfaccessory.TFAccessory
var doOnce sync.Once

// Startup is called by the platform management to start the platform up
func (p Platform) Startup(c *config.Config) platform.Control {
	p.Running = true
	return p
}

// Shutdown is called by the platform management to shut things down
func (p Platform) Shutdown() platform.Control {
	p.Running = false
	return p
}

// AddAccessory adds an envoy device, then adds it to HC
func (p Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	doOnce.Do(func() {
		envoys = make(map[string]*tfaccessory.TFAccessory)
	})

	e, err := envoy.New(a.IP)
	if err != nil {
		log.Info.Println(err.Error())
		return
	}
	settings, err := e.Info()

	a.Type = accessory.TypeSensor
	a.Info.Name = a.Name
	a.Info.SerialNumber = settings.Device.Sn
	a.Info.FirmwareRevision = settings.Device.Software
	a.Info.Model = "IQ Envoy"
	a.Info.Manufacturer = "Enphase"
	id, err := strconv.Atoi(settings.Device.Sn)
	if err != nil {
		a.Info.ID = 9999
	}
	a.Info.ID = uint64(id)

	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)

	envoys[a.IP] = a
	a.Device.(*devices.Envoy).Envoy = e
	log.Info.Printf("Enphase IQ Envoy ID: %s\n", a.Info.SerialNumber)

	// set initial state
	now, err := e.Now()
	if err != nil {
		log.Info.Println(err.Error())
		now = 0.0
	}
	if now == 0.0 {
		a.Device.(*devices.Envoy).Active.SetValue(characteristic.ActiveInactive)
		a.Device.(*devices.Envoy).LightSensor.CurrentAmbientLightLevel.SetValue(0.0001)
	} else {
		a.Device.(*devices.Envoy).LightSensor.CurrentAmbientLightLevel.SetValue(now)
	}
}

// GetAccessory looks up a device by IP address
func (p Platform) GetAccessory(ip string) (*tfaccessory.TFAccessory, bool) {
	val, ok := envoys[ip]
	return val, ok
}

// Background runs a background Go task periodically pinging everything
func (p Platform) Background() {
	go func() {
		for range time.Tick(time.Second * 300) {
			p.backgroundPuller()
		}
	}()
}

func (p Platform) backgroundPuller() {
	for _, a := range envoys {
		now, err := a.Device.(*devices.Envoy).Envoy.Now()
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		if now == 0.0 {
			a.Device.(*devices.Envoy).LightSensor.CurrentAmbientLightLevel.SetValue(0.0001)
			if a.Device.(*devices.Envoy).Active.GetValue() == characteristic.ActiveActive {
				a.Device.(*devices.Envoy).Active.SetValue(characteristic.ActiveInactive)
			}
		} else {
			// log.Info.Printf("Envoy Production: %f\n", now)
			a.Device.(*devices.Envoy).LightSensor.CurrentAmbientLightLevel.SetValue(now)
			if a.Device.(*devices.Envoy).Active.GetValue() == characteristic.ActiveInactive {
				a.Device.(*devices.Envoy).Active.SetValue(characteristic.ActiveActive)
			}
		}
	}
}
