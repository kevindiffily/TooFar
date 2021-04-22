package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

type Konnected struct {
	*accessory.Accessory

	SecuritySystem *KonnectedSvc
	Pins           map[uint8]interface{}
}

func NewKonnected(info accessory.Info) *Konnected {
	acc := Konnected{}
	acc.Accessory = accessory.New(info, accessory.TypeSecuritySystem)

	acc.SecuritySystem = NewKonnectedSvc()
	acc.AddService(acc.SecuritySystem.Service)
	acc.SecuritySystem.SecuritySystemCurrentState.SetValue(3) // default to Off

	alarmType := characteristic.NewSecuritySystemAlarmType()
	alarmType.SetValue(1)
	acc.SecuritySystem.AddCharacteristic(alarmType.Characteristic)

	acc.Pins = make(map[uint8]interface{})

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

type KonnectedContactSensor struct {
	*service.Service

	ContactSensorState *characteristic.ContactSensorState
	Name               *characteristic.Name
}

func NewKonnectedContactSensor(name string) *KonnectedContactSensor {
	svc := KonnectedContactSensor{}
	svc.Service = service.New(service.TypeContactSensor)

	svc.ContactSensorState = characteristic.NewContactSensorState()
	svc.AddCharacteristic(svc.ContactSensorState.Characteristic)

	svc.Name = characteristic.NewName()
	svc.Name.SetValue(name)
	svc.AddCharacteristic(svc.Name.Characteristic)

	return &svc
}

type KonnectedMotionSensor struct {
	*service.Service

	MotionDetected *characteristic.MotionDetected
	Name           *characteristic.Name
}

func NewKonnectedMotionSensor(name string) *KonnectedMotionSensor {
	svc := KonnectedMotionSensor{}
	svc.Service = service.New(service.TypeMotionSensor)

	svc.MotionDetected = characteristic.NewMotionDetected()
	svc.AddCharacteristic(svc.MotionDetected.Characteristic)

	svc.Name = characteristic.NewName()
	svc.Name.SetValue(name)
	svc.AddCharacteristic(svc.Name.Characteristic)

	return &svc
}

type KonnectedSystem struct {
	// not displayed in HC
	State bool
}
