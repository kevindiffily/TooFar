package runner

// this is distinct from toofar/action because of circular imports

import (
	"github.com/brutella/hc/log"
	"indievisible.org/toofar/action"
	"indievisible.org/toofar/platform"
)

func RunActions(as []*action.Action) {
	for _, a := range as {
		go runAction(a)
	}
}

func runAction(a *action.Action) {
	p, ok := platform.GetPlatform(a.TargetPlatform)
	log.Info.Printf("running action: %+v", a)
	if !ok {
		log.Info.Printf("unknown platform [%s]", a.TargetPlatform)
		return
	}
	d, ok := p.GetAccessory(a.TargetDevice)
	if !ok {
		log.Info.Printf("unknown device [%s]", a.TargetDevice)
		return
	}
	if d.Runner != nil {
		d.Runner(d, a)
	} else {
		log.Info.Printf("[%s] does not have an action runner", d.Name)
	}
}
