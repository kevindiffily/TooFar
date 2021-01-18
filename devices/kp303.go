package devices

import (
	"github.com/brutella/hc/accessory"
	// "github.com/brutella/hc/characteristic"
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
	}

	return &acc
}
