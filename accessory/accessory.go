package accessory

import (
	hcaccessory "github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
	// "github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"indievisible.org/toofar/action"
	"indievisible.org/toofar/devices"
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

	// its easier to just hang on to pointers to these than trying to build an interface{}...
	*hcaccessory.Switch
	*hcaccessory.ColoredLightbulb
	*hcaccessory.Lightbulb
	*hcaccessory.Thermometer
	*hcaccessory.Television
	*devices.HS220
	*devices.TXNR686
	*devices.TempLightbulb
	*service.HumiditySensor
	*service.BridgingState

	Actions []action.Action
	Runner  func(*TFAccessory, *action.Action)
}

// MatchActions returns a slice of actions that should be run
// jumping through hoops since including platform here would be circular
func (a TFAccessory) MatchActions(state string) []*action.Action {
	log.Info.Printf("MatchActions: %s", state)
	actions := make([]*action.Action, 0)
	for _, action := range a.Actions {
		if action.TriggerState == state {
			log.Info.Printf("%+v", action)
			actions = append(actions, &action)
		}
	}
	return actions
}
