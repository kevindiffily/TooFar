package devices

import (
	"github.com/brutella/hc/accessory"
)

// a system might have several chips
type Noonlight struct {
	*accessory.Accessory
}

func NewNoonlight(info accessory.Info) *Noonlight {
	acc := Noonlight{}
	acc.Accessory = accessory.New(info, accessory.TypeSensor)
	return &acc
}
