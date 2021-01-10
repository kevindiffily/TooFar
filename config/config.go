package config

import (
	"github.com/brutella/hc"
)

// Config is the primary daemon configuration...
type Config struct {
	ConfigDir   string    // passed in from CLI
	ConfigFile  string    // server.json
	HTTPAddress string    // net.Dial address format, :port is good enough
	Name        string    // what this bridge shows as
	ID          string    // displayed serial number
	HCConfig    hc.Config // base HomeControl configuration
}
