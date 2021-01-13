package toofar

import (
	"fmt"
	"github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/homecontrol"
	"github.com/cloudkucooland/toofar/kasa"
	"github.com/cloudkucooland/toofar/linuxsensors"
	"github.com/cloudkucooland/toofar/onkyo"
	"github.com/cloudkucooland/toofar/owm"
	"github.com/cloudkucooland/toofar/ping"
	"github.com/cloudkucooland/toofar/platform"
	"github.com/cloudkucooland/toofar/shelly"
	"github.com/cloudkucooland/toofar/tfhttp"
	"github.com/cloudkucooland/toofar/tradfri"
	"time"

	"github.com/brutella/hc/log"
)

// BootstrapPlatforms sets up all the platforms
// hardcode this until I can spend the time to make it dynamic
func BootstrapPlatforms(c *config.Config) {
	config.Set(c)

	var h tfhttp.Platform
	platform.RegisterPlatform("HTTP", h)

	var s shelly.Platform
	platform.RegisterPlatform("Shelly", s)

	var tp tradfri.Platform
	platform.RegisterPlatform("Tradfri", tp)

	var kp kasa.Platform
	platform.RegisterPlatform("Kasa", kp)

	var owmp owm.Platform
	platform.RegisterPlatform("OWM", owmp)

	var onkp onkyo.Platform
	platform.RegisterPlatform("Onkyo", onkp)

	var png ping.Platform
	platform.RegisterPlatform("Ping", png)

	var ls linuxsensors.Platform
	platform.RegisterPlatform("LinuxSensors", ls)

	var hcp tfhc.HCPlatform
	platform.RegisterPlatform("HomeControl", hcp)

	platform.StartupAllPlatforms(c)

	// add a sensor
	sensor := accessory.TFAccessory{}
	ls.AddAccessory(&sensor)

	// auto-discover Kasa devices
	if c.Discover {
		kp.Discover() // UDP probe for Kasa devices
		// s.Discover() // TBD shelly discovery
		// wait a bit for discovery to complete -- otherwise HCStart runs before all devices are found
		time.Sleep(time.Second * 2)
	}
}

// AddAccessory is a wrapper to each platform's AddAccessory, no need to expose each platform to the daemon
func AddAccessory(h *accessory.TFAccessory) error {
	if h.Platform == "" {
		err := fmt.Errorf("accessory platform unset: %+v", h)
		log.Info.Print(err)
		return err
	}

	p, ok := platform.GetPlatform(h.Platform)
	if !ok {
		err := fmt.Errorf("unknown accessory platform: %+v", h)
		log.Info.Print(err)
		return err
	}

	p.AddAccessory(h)
	return nil
}

// StartHC is just a wrapper, no need to expose tfhc to the daemon
func StartHC() {
	tfhc.StartHC()
}
