package devices

import (
	"github.com/brutella/hc/accessory"
	// "github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

type Konnected struct {
	*accessory.Accessory

	Mac            string
	SecuritySystem *KonnectedSvc
}

func NewKonnected(info accessory.Info) *Konnected {
	acc := Konnected{}
	acc.Accessory = accessory.New(info, accessory.TypeSecuritySystem)

	acc.SecuritySystem = NewKonnectedSvc()
	acc.AddService(acc.SecuritySystem.Service)
	return &acc
}

type KonnectedSvc struct {
	*service.Service

	// On         *characteristic.On
}

func NewKonnectedSvc() *KonnectedSvc {
	svc := KonnectedSvc{}
	svc.Service = service.New(service.TypeSecuritySystem)

	// svc.On = characteristic.NewOn()
	// svc.AddCharacteristic(svc.On.Characteristic)

	return &svc
}
