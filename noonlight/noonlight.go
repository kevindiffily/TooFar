package noonlight

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"
)

// Platform is the platform handle for the Noonlight stuff
type Platform struct {
	noonlight *tfaccessory.TFAccessory
}

// Startup is called by the platform management to start the platform up
func (p Platform) Startup(c *config.Config) platform.Control {
	// nothing to do
	return p
}

// Shutdown is called by the platform management to shut things down
func (p Platform) Shutdown() platform.Control {
	// nothing to do
	return p
}

// AddAccessory adds an envoy device, then adds it to HC
func (p Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	if p.noonlight != nil {
		log.Info.Printf("noonlight already configured, ignoring new config; using: %+v", p.noonlight)
		return
	}

	a.Type = accessory.TypeSensor
	a.Info.Name = a.Name
	a.Info.SerialNumber = "1234567"
	a.Info.FirmwareRevision = "0.0"
	a.Info.Model = "noonlight for TooFar"
	a.Info.Manufacturer = "TooFar"
	a.Info.ID = 1234567

	a.Device = devices.NewNoonlight(a.Info)
	a.Accessory = a.Device.(*devices.Noonlight).Accessory

	a.Runner = runner

	p.noonlight = a

	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)
}

// GetAccessory returns the single noonlight accessory
func (p Platform) GetAccessory(name string) (*tfaccessory.TFAccessory, bool) {
	return p.noonlight, true
}

func (p Platform) Background() {
	// nothing
}

func runner(a *tfaccessory.TFAccessory, m *action.Action) {
	log.Info.Println("noonlight action runner")
}
