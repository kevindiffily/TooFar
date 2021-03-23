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
	Pins           map[uint8]*KonnectedPinSvc
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

	acc.Pins = make(map[uint8]*KonnectedPinSvc)

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

	svc.SecuritySystemCurrentState = characteristic.NewSecuritySystemCurrentState()
	svc.AddCharacteristic(svc.SecuritySystemCurrentState.Characteristic)

	svc.SecuritySystemTargetState = characteristic.NewSecuritySystemTargetState()
	svc.AddCharacteristic(svc.SecuritySystemTargetState.Characteristic)

	return &svc
}

type KonnectedPinSvc struct {
	*service.Service

	ContactSensorState *characteristic.ContactSensorState
	Name               *characteristic.Name
}

func NewKonnectedPinSvc(name string) *KonnectedPinSvc {
	svc := KonnectedPinSvc{}
	svc.Service = service.New(service.TypeContactSensor)

	svc.ContactSensorState = characteristic.NewContactSensorState()
	svc.AddCharacteristic(svc.ContactSensorState.Characteristic)

	svc.Name = characteristic.NewName()
	svc.Name.SetValue(name)
	svc.AddCharacteristic(svc.Name.Characteristic)

	return &svc
}
