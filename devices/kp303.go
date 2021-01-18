package devices

import (
	"github.com/brutella/hc/accessory"
	// "github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

type KP303 struct {
	*accessory.Accessory
	One   *service.Outlet
	Two   *service.Outlet
	Three *service.Outlet
}

func NewKP303(info accessory.Info) *KP303 {
	acc := KP303{}
	acc.Accessory = accessory.New(info, accessory.TypeLightbulb)

	acc.One = service.NewOutlet()
	acc.AddService(acc.One.Service)
	acc.Two = service.NewOutlet()
	acc.AddService(acc.Two.Service)
	acc.Three = service.NewOutlet()
	acc.AddService(acc.Three.Service)

	return &acc
}
