package accessory

import (
	hcaccessory "github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	"github.com/cloudkucooland/toofar/action"
)

// TFAccessory is the accessory type, TooFar's stuff, plus hc's stuff
type TFAccessory struct {
	Platform string // Kasa, Tradfri, Tradfri-Device, Shelly, etc
	Name     string // the name used internally
	// the accessory's config file name or dynamically determined for discovered devices
	IP       string // the IP address of the device (shelly, kasa, tradfri)
	Username string // for Tradfri, Shelly
	Password string // for Tradfri, Shelly
	// do we still need this?
	Type hcaccessory.AccessoryType // defined at https://github.com/brutella/hc/tree/master/accessory

	// embedded struct (pointer)
	Info                   hcaccessory.Info // defined at https://github.com/brutella/hc/blob/master/accessory/accessory.go
	*hcaccessory.Accessory                  // set when the device is added to HomeControl

	Device interface{}

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
