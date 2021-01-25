package tfhc

import (
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"
	"github.com/cloudkucooland/toofar/runner"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/util"
	// "github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	// "github.com/brutella/hc/service"
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
		FirmwareRevision: "0.0.9",
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

// AddAccessory registers a device with HC and does basic setup
// MOST devices will be set up by their own platform's AddAccessory()
// the various platforms call this after doing their work if a UI is needed
func (h HCPlatform) AddAccessory(a *tfaccessory.TFAccessory) {
	switch a.Type {
	case accessory.TypeSwitch:
		switch a.Info.Model {
		case "KP303(US)":
			kp := devices.NewKP303(a.Info)
			a.Device = kp
			a.Accessory = kp.Accessory
			a.Runner = genericSwitchActionRunner
			for i := 0; i < len(kp.Outlets); i++ {
				kp.Outlets[i].On.OnValueRemoteUpdate(func(newval bool) {
					if newval {
						actions := a.MatchActions("On")
						runner.RunActions(actions)
					} else {
						actions := a.MatchActions("Off")
						runner.RunActions(actions)
					}
				})
			}
		case "onkyo-controller":
			log.Info.Println("adding onkyo-controller")
			oc := devices.NewOnkyoController(a.Info)
			a.Device = oc
			a.Accessory = oc.Accessory
		default:
			d := accessory.NewSwitch(a.Info)
			a.Device = d
			a.Accessory = d.Accessory
			a.Runner = genericSwitchActionRunner
			d.Switch.On.OnValueRemoteUpdate(func(newval bool) {
				if newval {
					actions := a.MatchActions("On")
					runner.RunActions(actions)
				} else {
					actions := a.MatchActions("Off")
					runner.RunActions(actions)
				}
			})
		}
	case accessory.TypeLightbulb:
		switch a.Info.Model {
		case "TRADFRI bulb E26 CWS opal 600lm":
			clb := accessory.NewColoredLightbulb(a.Info)
			a.Device = clb
			a.Accessory = clb.Accessory
		case "HS220(US)":
			hs := devices.NewHS220(a.Info)
			a.Device = hs
			a.Accessory = hs.Accessory
		case "TRADFRI bulb E26 WS opal 980lm":
			tlb := devices.NewTempLightbulb(a.Info)
			a.Device = tlb
			a.Accessory = tlb.Accessory
		case "LTD010":
			tlb := devices.NewTempLightbulb(a.Info)
			a.Device = tlb
			a.Accessory = tlb.Accessory
		default:
			log.Info.Printf("unknown lightbulb type, using generic: [%s]", a.Info.Model)
			lb := accessory.NewLightbulb(a.Info)
			a.Device = lb
			a.Accessory = lb.Accessory
		}
	case accessory.TypeSensor:
		switch a.Info.Model {
		case "OS Sensors":
			a.Device = devices.NewLinuxSensors(a.Info)
			a.Accessory = a.Device.(*devices.LinuxSensors).Accessory
		default:
			a.Device = accessory.NewTemperatureSensor(a.Info, 20, -10, 55, 0.1)
			a.Accessory = a.Device.(*accessory.Thermometer).Accessory
		}
	/* case accessory.TypeSecuritySystem:
	a.Accessory = accessory.New(a.Info, a.Type) */
	case accessory.TypeTelevision:
		switch a.Info.Model {
		case "TX-NR686":
			tx := devices.NewTXNR686(a.Info)
			a.Device = tx
			a.Accessory = tx.Accessory
		default:
			log.Info.Printf("unknown television type, using generic: [%s]", a.Info.Model)
			tv := accessory.NewTelevision(a.Info)
			a.Device = tv
			a.Accessory = tv.Accessory
		}
	default:
		log.Info.Printf("unknown accessory type, using generic: [%s]", a.Info.Model)
		a.Accessory = accessory.New(a.Info, a.Type)
		a.Device = a.Accessory
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

	log.Info.Printf("Added: %T: %+v", a.Device, a.Device)

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

func genericSwitchActionRunner(a *tfaccessory.TFAccessory, action *action.Action) {
	log.Info.Printf("generic switch action runner: %+v %+v", a, action)
}
