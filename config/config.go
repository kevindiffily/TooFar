package config

import (
	"github.com/brutella/hc"
)

// Config is the primary daemon configuration...
type Config struct {
	ConfigDir      string    // passed in from CLI
	ConfigFile     string    // server.json
	HTTPAddress    string    // net.Dial address format, :port is good enough
	Name           string    // what this bridge shows as
	ID             string    // displayed serial number -- if you run multiple instances, make sure each has a distinct ID
	HCConfig       hc.Config // base HomeControl configuration
	Discover       bool      // run Kasa & Shelly auto-discovery (does not work properly yet, do not enable)
	KasaPullRate   int       // (seconds) how frequently to pull Kasa devices -- 0 to disable
	ShellyPullRate int       // seconds - 0 to disable (TBD)
}

var runningConfig *Config

// Get a pointer to the global config
func Get() *Config {
	return runningConfig
}

// should only be called by the bootstrap
func Set(c *Config) {
	runningConfig = c
}
