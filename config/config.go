package config

import (
	"github.com/brutella/hc"
)

// Config is the global daemon configuration...
type Config struct {
	ConfigDir         string    // passed in from CLI
	ConfigFile        string    // server.json
	HTTPAddress       string    // net.Dial address format, :port is good enough
	Name              string    // what this bridge shows as
	ID                string    // displayed serial number -- if you run multiple instances, make sure each has a distinct ID
	HCConfig          hc.Config // base HomeControl configuration
	Discover          bool      // run Kasa, Konnected, & Shelly auto-discovery (does not work properly yet, do not enable)
	KasaPullRate      uint16    // (seconds) how frequently to pull Kasa devices -- 0 to disable
	KasaBroadcasts    uint8     // number of UDP broadcast packets to send - 1 is usually enough -- (unset/0/1 sends 1 packet)
	KasaTimeout       uint8     // how long to wait for direct (TCP) pulls
	ShellyPullRate    uint16    // 0 to disable pulling
	ShellyTimeout     uint8     // how long to wait for direct pulls
	KonnectedTimeout  uint8     /// how long to wait for direct pulls
	KonnectedPullRate uint16    /// how frequently to pull , 0 to disable
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
