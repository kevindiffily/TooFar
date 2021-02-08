package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

type KP303 struct {
	*accessory.Accessory
	Outlets []*service.Outlet
}

func NewKP303(info accessory.Info) *KP303 {
	acc := KP303{}
	acc.Accessory = accessory.New(info, accessory.TypeLightbulb)

	acc.Outlets = make([]*service.Outlet, 3, 4)
	for i := 0; i < 3; i++ {
		acc.Outlets[i] = service.NewOutlet()
		acc.AddService(acc.Outlets[i].Service)

		pm := characteristic.NewProgramMode()
		acc.Outlets[i].AddCharacteristic(pm.Characteristic)
		pm.SetValue(characteristic.ProgramModeNoProgramScheduled)

		sd := characteristic.NewSetDuration()
		acc.Outlets[i].AddCharacteristic(sd.Characteristic)

		rd := characteristic.NewRemainingDuration()
		acc.Outlets[i].AddCharacteristic(rd.Characteristic)
		rd.SetValue(0)
	}

	return &acc
}
