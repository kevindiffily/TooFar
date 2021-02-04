package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/service"
)

type Konnected struct {
	*accessory.Accessory

	SecuritySystem *KonnectedSvc
}

func NewKonnected(info accessory.Info) *Konnected {
	acc := Konnected{}
	acc.Accessory = accessory.New(info, accessory.TypeSecuritySystem)

	acc.SecuritySystem = NewKonnectedSvc()
	acc.AddService(acc.SecuritySystem.Service)
	acc.SecuritySystem.SecuritySystemTargetState.OnValueRemoteUpdate(func(newval int) {
		// do the work to adjust the state
		log.Info.Printf("HC requested system state change to %d", newval)
		acc.SecuritySystem.SecuritySystemCurrentState.SetValue(newval)
	})

	alarmType := characteristic.NewSecuritySystemAlarmType()
	alarmType.SetValue(1)
	acc.SecuritySystem.AddCharacteristic(alarmType.Characteristic)

	return &acc
}

type KonnectedSvc struct {
	*service.Service

	SecuritySystemCurrentState *characteristic.SecuritySystemCurrentState
	SecuritySystemTargetState  *characteristic.SecuritySystemTargetState
}

func NewKonnectedSvc() *KonnectedSvc {
	svc := KonnectedSvc{}
	svc.Service = service.New(service.TypeSecuritySystem)

	characteristic.NewSecuritySystemCurrentState()
	svc.AddCharacteristic(svc.SecuritySystemCurrentState.Characteristic)

	svc.SecuritySystemTargetState = characteristic.NewSecuritySystemTargetState()
	svc.AddCharacteristic(svc.SecuritySystemTargetState.Characteristic)

	return &svc
}
