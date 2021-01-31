package onkyo

import (
	"github.com/cloudkucooland/go-eiscp"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"

	"fmt"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	"math"
	"strconv"
)

func addController(parent *tfaccessory.TFAccessory) {
	a := tfaccessory.TFAccessory{}
	a.Type = accessory.TypeSwitch
	a.Name = fmt.Sprintf("%s-controller", parent.Info.Model)

	a.Info.Manufacturer = parent.Info.Manufacturer
	a.Info.Model = "onkyo-controller"
	a.Info.SerialNumber = fmt.Sprintf("%s-controller", parent.Info.SerialNumber)
	a.Info.FirmwareRevision = parent.Info.FirmwareRevision
	a.Info.Name = a.Name
	a.Info.ID = 86404

	// add to HC for GUI
	log.Info.Printf("adding [%s]", a.Info.Name)
	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(&a)

	// music optimizer toggle - not correct at bridge start yet
	oc := a.Device.(*devices.OnkyoController)
	onkyo := parent.Device.(*devices.TXNR686)
	oc.MusicOptimizer.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("Setting music optimizer to %t", newstate)
		mot := "00"
		if newstate {
			mot = "01"
		}
		onkyo.Amp.SetOnly("MOT", mot)
	})

	// front panel brightness
	dimName, err := onkyo.Amp.GetDimmer()
	if err != nil {
		log.Info.Println(err.Error())
		dimName = "Bright"
	}
	dimCode, ok := eiscp.DimmerState[dimName]
	var dimmer int
	if ok {
		t, _ := strconv.ParseInt(dimCode, 16, 8)
		dimmer = int(t * 34)
	}
	oc.Dimmer.Value.SetValue(dimmer)
	oc.Dimmer.Value.OnValueRemoteUpdate(func(newstate int) {
		neg := float64(2 - (newstate / 34))
		pos := uint8(math.Abs(neg))
		onkyo.Amp.SetOnly("DIM", fmt.Sprintf("%02d", pos))
	})

	// master volume level
	mvl, err := onkyo.Amp.GetVolume()
	if err != nil {
		log.Info.Println(err.Error())
		mvl = 50
	}
	oc.Volume.Value.SetValue(int(mvl))
	oc.Volume.Value.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("Setting volume to %d", newstate)
		onkyo.Amp.SetVolume(uint8(newstate))
	})
}
