package platform

import (
	// "fmt"
	// "github.com/brutella/hc/log"
	"indievisible.org/toofar/accessory"
	"indievisible.org/toofar/config"
	"sync"
)

// Control is the interface which all platforms must satisfy
type Control interface {
	Startup(config.Config) Control
	Background()
	Shutdown() Control
	AddAccessory(*accessory.TFAccessory)
	GetAccessory(string) (*accessory.TFAccessory, bool)
}

var platforms map[string]Control
var doOnce sync.Once

// RegisterPlatform is called whenever a new platform is instantiated
func RegisterPlatform(name string, control Control) {
	doOnce.Do(func() {
		platforms = make(map[string]Control)
	})
	if !platformExists(name) {
		platforms[name] = control
	}
}

// GetPlatform looks up a registered platform by name
func GetPlatform(name string) (Control, bool) {
	pc, ok := platforms[name]
	return pc, ok
}

func platformExists(name string) bool {
	_, ok := platforms[name]
	return ok
}

// ShutdownAllPlatforms is called at process stop to shutdown all platforms
func ShutdownAllPlatforms() {
	for name, platform := range platforms {
		// log.Info.Printf("Shutting down: %s", name)
		platforms[name] = platform.Shutdown()
	}
}

// StartupAllPlatforms is called at process start to initialize all platforms
func StartupAllPlatforms(c config.Config) {
	for name, platform := range platforms {
		// log.Info.Printf("Starting up: %s", name)
		platforms[name] = platform.Startup(c)
	}
}

// Background starts the background processes for every process
func Background() {
	for _, platform := range platforms {
		// log.Info.Printf("Starting background processes: %s", name)
		platform.Background()
	}
}
