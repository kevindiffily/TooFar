package samsung

import (
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	// "github.com/cloudkucooland/toofar/devices"
	"github.com/brutella/hc/log"
)

func runner(a *tfaccessory.TFAccessory, d *action.Action) {
	// dev := a.Device.(*devices.SamsungTV)
	switch d.Verb {
	case "Stop":
		log.Info.Printf("called stop")
	default:
		log.Info.Printf("unknown verb %s (valid: Stop)", d.Verb)
	}
}
