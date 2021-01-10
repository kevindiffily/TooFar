package devices

import (
	"github.com/brutella/hc/accessory"
	// "github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

type StatelessSwitch struct {
	*accessory.Accessory
	StatelessSwitch *service.StatelessProgrammableSwitch
}

func NewStatelessSwitch(info accessory.Info) *StatelessSwitch {
	acc := StatelessSwitch{}
	acc.Accessory = accessory.New(info, accessory.TypeProgrammableSwitch)
	acc.StatelessSwitch = service.NewStatelessProgrammableSwitch()
	acc.AddService(acc.StatelessSwitch.Service)

	return &acc
}
