package accessory

import (
	hcaccessory "github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
	// "github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/cloudkucooland/toofar/action"
	// "github.com/cloudkucooland/toofar/devices"
)

// TFAccessory is the accessory type, TooFar's stuff, plus hc's stuff
type TFAccessory struct {
	Platform string // Kasa, Tradfri, Tradfri-Device, Shelly, etc
	Name     string // the name used internally
	// the accessory's config file name or dynamically determined for discovered devices
	IP       string                    // the IP address of the device (shelly, kasa, tradfri)
	Username string                    // for Tradfri, Shelly
	Password string                    // for Tradfri, Shelly
	Type     hcaccessory.AccessoryType // defined at https://github.com/brutella/hc/tree/master/accessory

	// embedded struct (pointer)
	Info                   hcaccessory.Info // defined at https://github.com/brutella/hc/blob/master/accessory/accessory.go
	*hcaccessory.Accessory                  // set when the device is added to HomeControl

	// move these to the Device interface{}... now that things are a bit more stable
	// *devices.TXNR686
	// *devices.OnkyoController

	Device interface{}

	// make distinct OpenWeatherMap device type and move this there
	*service.HumiditySensor

	Actions []action.Action
	Runner  func(*TFAccessory, *action.Action)
}

// MatchActions returns a slice of actions that should be run
// jumping through hoops since including platform here would be circular
func (a TFAccessory) MatchActions(state string) []*action.Action {
	log.Info.Printf("MatchActions: %s", state)
	var actions []*action.Action
	for _, action := range a.Actions {
		if action.TriggerState == state {
			log.Info.Printf("%s: %+v", action.TriggerState, action)
			actions = append(actions, &action)
		}
	}
	log.Info.Printf("%+v", actions)
	return actions
}
