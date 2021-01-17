package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
)

// a system might have several chips
type LinuxSensors struct {
	*accessory.Accessory
	Chips map[string]*SensorChipValues
}

// each chip might have several temps or fans
type SensorChipValues map[string]*service.TemperatureSensor

func NewLinuxSensors(info accessory.Info) *LinuxSensors {
	acc := LinuxSensors{}
	acc.Accessory = accessory.New(info, accessory.TypeSensor)

	acc.Chips = make(map[string]*SensorChipValues)
	return &acc
}
