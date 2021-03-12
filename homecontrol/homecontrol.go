package tfhc

import (
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/platform"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/util"
	"sync"
)

// HCPlatform is the platform handle
type HCPlatform struct {
	Running bool
}

var doOnceHC sync.Once
var hcs map[string]*tfaccessory.TFAccessory

// Startup is called by the platform bootstrap
func (h HCPlatform) Startup(c *config.Config) platform.Control {
	doOnceHC.Do(func() {
		hcs = make(map[string]*tfaccessory.TFAccessory)
		h.Running = true
	})
	return h
}

// StartHC is called after all devices are discovered/registered to start operation
func StartHC() {
	c := config.Get()
	storage, err := util.NewFileStorage("serials")
	if err != nil {
		log.Info.Println("unable to get storage")
	}
	serial := util.GetSerialNumberForAccessoryName("TooFarRoot", storage)

	if c.Name == "" {
		c.Name = "TooFar"
	}
	root := accessory.NewBridge(accessory.Info{
		Name:             c.Name,
		ID:               1,
		SerialNumber:     serial,
		Manufacturer:     "deviousness",
		Model:            "TooFar",
		FirmwareRevision: "0.0.10",
	})
	root.Accessory.OnIdentify(func() {
		log.Info.Printf("bridge root identify called: %+v", root.Accessory)
	})

	// all the other registered things
	values := []*accessory.Accessory{}
	for _, v := range hcs {
		values = append(values, v.Accessory)
	}
	config := hc.Config(c.HCConfig)
	// var err error
	transport, err := hc.NewIPTransport(config, root.Accessory, values...)
	if err != nil {
		log.Info.Panic(err)
	}

	hc.OnTermination(func() {
		<-transport.Stop()
	})
	go transport.Start()
	uri, _ := transport.XHMURI()
	log.Info.Printf("add this bridge with: %s", uri)
}

// Shutdown is called at process teardown
func (h HCPlatform) Shutdown() platform.Control {
	h.Running = false
	return h
}

// AddAccessory registers a device with HC
func (h HCPlatform) AddAccessory(a *tfaccessory.TFAccessory) {
	// catch devices that didn't get migrated properly
	if a.Accessory == nil {
		log.Info.Printf("accessory unset: %v", a.Info)
		return
	}

	a.Accessory.OnIdentify(func() {
		log.Info.Printf("identify called for [%s]: %+v", a.Name, a.Accessory)
		for _, service := range a.Accessory.GetServices() {
			log.Info.Printf("service: %+v", service)
			for _, char := range service.GetCharacteristics() {
				log.Info.Printf("characteristic : %+v", char)
			}
		}
	})

	hcs[a.Name] = a
}

// GetAccessory looks up a device by name -- you probably want the various platform's version, not this
func (h HCPlatform) GetAccessory(name string) (*tfaccessory.TFAccessory, bool) {
	a, ok := hcs[name]
	return a, ok
}

// Background runs the various background tasks: none for HC
func (h HCPlatform) Background() {
	// pull the dummy switches and confirm their state?
}
