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
	"github.com/brutella/hc/service"
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
		a.Switch = accessory.NewSwitch(a.Info)
		a.Accessory = a.Switch.Accessory
		a.Runner = genericSwitchActionRunner
		a.Switch.Switch.On.OnValueRemoteUpdate(func(newval bool) {
			if newval {
				actions := a.MatchActions("On")
				runner.RunActions(actions)
			} else {
				actions := a.MatchActions("Off")
				runner.RunActions(actions)
			}
		})
	case accessory.TypeProgrammableSwitch:
		a.StatelessSwitch = devices.NewStatelessSwitch(a.Info)
		a.Accessory = a.StatelessSwitch.Accessory
		a.StatelessSwitch.StatelessSwitch.ProgrammableSwitchEvent.SetValue(0)
		a.Runner = statelessSwitchActionRunner
		a.StatelessSwitch.StatelessSwitch.ProgrammableSwitchEvent.OnValueRemoteUpdate(func(newval int) {
			log.Info.Printf("running stateless switch: %d", newval)
			if newval == 0 {
				// actions := a.MatchActions("On")
				// runner.RunActions(actions)
			} else {
				// actions := a.MatchActions("Off")
				// runner.RunActions(actions)
			}
		})
	case accessory.TypeLightbulb:
		switch a.Info.Model {
		case "TRADFRI bulb E26 CWS opal 600lm":
			a.ColoredLightbulb = accessory.NewColoredLightbulb(a.Info)
			a.Accessory = a.ColoredLightbulb.Accessory
		case "HS220(US)":
			a.HS220 = devices.NewHS220(a.Info)
			a.Accessory = a.HS220.Accessory
		case "TRADFRI bulb E26 WS opal 980lm":
			a.TempLightbulb = devices.NewTempLightbulb(a.Info)
			a.Accessory = a.TempLightbulb.Accessory
		case "LTD010":
			a.TempLightbulb = devices.NewTempLightbulb(a.Info)
			a.Accessory = a.TempLightbulb.Accessory
		default:
			log.Info.Printf("unknown lightbulb type, using generic: [%s]", a.Info.Model)
			a.Lightbulb = accessory.NewLightbulb(a.Info)
			a.Accessory = a.Lightbulb.Accessory
		}
	case accessory.TypeSensor:
		a.Thermometer = accessory.NewTemperatureSensor(a.Info, 20, -10, 45, 1)
		a.Accessory = a.Thermometer.Accessory
	case accessory.TypeSecuritySystem:
		a.Accessory = accessory.New(a.Info, a.Type)
	case accessory.TypeTelevision:
		switch a.Info.Model {
		case "TX-NR686":
			a.TXNR686 = devices.NewTXNR686(a.Info)
			a.Accessory = a.TXNR686.Accessory
		default:
			a.Television = accessory.NewTelevision(a.Info)
			a.Accessory = a.Television.Accessory
		}
	default:
		a.Accessory = accessory.New(a.Info, a.Type)
	}

	// deprecated by HomeKit; still visible in Control.app
	a.BridgingState = service.NewBridgingState()
	a.Accessory.AddService(a.BridgingState.Service)
	a.BridgingState.Reachable.SetValue(true)
	a.BridgingState.Reachable.Description = "BridgingState.Reachable"
	// a.BridgingState.LinkQuality.SetValue(1)
	a.BridgingState.LinkQuality.Description = "BridgingState.LinkQuality"
	a.BridgingState.AccessoryIdentifier.SetValue(a.Name)
	a.BridgingState.AccessoryIdentifier.Description = "BridgingState.AccessoryIdentifier"
	// a.BridgingState.Category.SetValue(1)
	a.BridgingState.Category.Description = "BridgingState.Category"

	a.Accessory.OnIdentify(func() {
		log.Info.Printf("identify called for [%s]: %+v", a.Name, a.Accessory)
		for _, service := range a.Accessory.GetServices() {
			log.Info.Printf("service: %+v", service)
			for _, char := range service.GetCharacteristics() {
				log.Info.Printf("characteristic : %+v", char)
			}
		}
	})
	// the other platforms keep their own pointers indexed as they need
	// this is only ever used for "dummy" devices which have no hardware associated
	// with another platform.
	hcs[a.Name] = a
}

// GetAccessory looks up a device by name -- you probably want the various platform's version, not this
func (h HCPlatform) GetAccessory(name string) (*tfaccessory.TFAccessory, bool) {
	a, ok := hcs[name]
	return a, ok
}

// Background runs the various background tasks: none for HC
func (h HCPlatform) Background() {
	// nothing
}

func genericSwitchActionRunner(a *tfaccessory.TFAccessory, action *action.Action) {
	log.Info.Printf("generic switch action runner: %+v %+v", a, action)
}

func statelessSwitchActionRunner(a *tfaccessory.TFAccessory, action *action.Action) {
	log.Info.Printf("stateless switch action runner: %+v %+v", a, action)
}
