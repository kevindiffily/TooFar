package onkyo

import (
	// "encoding/json"
	"github.com/cloudkucooland/go-eiscp"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	"github.com/cloudkucooland/toofar/devices"

	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"strconv"
	"time"
)

func runner(a *tfaccessory.TFAccessory, d *action.Action) {
	// log.Info.Printf("in onkyo action runner: %+v", d)
	dev := a.Device.(*devices.OnkyoReceiver)
	switch d.Verb {
	case "Stop":
		log.Info.Printf("called stop")
		dev.Amp.SetNetworkPlayStatus("Sxx")
		// dev.Television.Active.SetValue(characteristic.ActiveInactive)
		// dev.VolumeActive.SetValue(characteristic.ActiveInactive)
	case "TuneInPreset":
		// http://vtochq-it.blogspot.com/2018/12/onkyo-pioneer-network-remote-control.html
		// log.Info.Printf("setting preset to %s", d.Value)
		if dev.Television.On.GetValue() {
			dev.Amp.SetPower(true)
			dev.Television.On.SetValue(true)
			dev.Television.Active.SetValue(characteristic.ActiveActive)
			dev.VolumeActive.SetValue(characteristic.ActiveActive)
			time.Sleep(time.Second)
		}

		source, err := dev.Amp.GetSource()
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		if source != eiscp.SrcNetwork {
			_, err := dev.Amp.SetSource(eiscp.SrcNetwork)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			time.Sleep(time.Second)
		}

		log.Info.Println("setting to tuneIN radio")
		err = dev.Amp.SetNetworkServiceTuneIn()
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		time.Sleep(time.Second * 3)
		log.Info.Println("setting presets screen")
		err = dev.Amp.SelectNetworkListItem(1) // first item in the list is "Presets"
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		time.Sleep(time.Second * 3)
		log.Info.Printf("setting to selected preset: %s", d.Value)
		pi, err := strconv.Atoi(d.Value)
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		err = dev.Amp.SelectNetworkListItem(pi)
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
	default:
		log.Info.Printf("unknown verb %s (valid: TuneInPreset, Stop)", d.Verb)
	}
}
