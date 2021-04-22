package runner

// this is distinct from toofar/action because of circular imports

import (
	"github.com/brutella/hc/log"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	"github.com/cloudkucooland/toofar/platform"
)

func RunActions(as []action.Action) {
	for _, a := range as {
		go runAction(&a)
	}
}

func runAction(a *action.Action) {
	p, ok := platform.GetPlatform(a.TargetPlatform)
	// log.Info.Printf("running action: %+v", a)
	if !ok {
		log.Info.Printf("unknown platform [%s]", a.TargetPlatform)
		return
	}
	d, ok := p.GetAccessory(a.TargetDevice)
	if !ok {
		log.Info.Printf("unknown device [%s]", a.TargetDevice)
		return
	}
	if d.Runner == nil {
		log.Info.Printf("[%s] does not have an action runner", d.Name)
		return
	}
	d.Runner(d, a)
}

func GenericSwitchActionRunner(a *tfaccessory.TFAccessory, action *action.Action) {
	log.Info.Printf("generic switch action runner: %+v %+v", a, action)
}
