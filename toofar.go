package toofar

import (
	"fmt"
	"indievisible.org/toofar/accessory"
	"indievisible.org/toofar/config"
	"indievisible.org/toofar/homecontrol"
	"indievisible.org/toofar/kasa"
	"indievisible.org/toofar/onkyo"
	"indievisible.org/toofar/owm"
	"indievisible.org/toofar/ping"
	"indievisible.org/toofar/platform"
	"indievisible.org/toofar/shelly"
	"indievisible.org/toofar/tfhttp"
	"indievisible.org/toofar/tradfri"

	"github.com/brutella/hc/log"
)

// BootstrapPlatforms sets up all the platforms
// hardcode this until I can spend the time to make it dynamic
func BootstrapPlatforms(c config.Config) {
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

	var hcp tfhc.HCPlatform
	platform.RegisterPlatform("HomeControl", hcp)

	platform.StartupAllPlatforms(c)
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
func StartHC(c config.Config) {
	tfhc.StartHC(c)
}
