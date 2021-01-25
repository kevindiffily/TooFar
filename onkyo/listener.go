package onkyo

import (
	// "encoding/json"
	"github.com/cloudkucooland/go-eiscp"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/devices"

	"fmt"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"strconv"
)

func iscpListener(a *tfaccessory.TFAccessory) {
	o := a.Device.(*devices.TXNR686)
	for resp := range o.Amp.Responses {
		v := resp.Parsed
		switch resp.Command {
		case "PWR":
			if o.Television.On.GetValue() != v.(bool) {
				p := characteristic.ActiveInactive
				if v.(bool) {
					p = characteristic.ActiveActive
				}
				o.Television.On.SetValue(v.(bool))
				o.Television.Active.SetValue(p)
				o.VolumeActive.SetValue(p) // speaker
			}
		case "MVL":
			if int(v.(uint8)) != o.Television.Volume.GetValue() {
				o.Television.Volume.SetValue(int(v.(uint8)))
			}
		case "AMT":
			if v.(bool) != o.Speaker.Mute.GetValue() {
				o.Speaker.Mute.SetValue(v.(bool))
			}
		case "TPD":
			if float64(v.(int8)) != o.Temp.CurrentTemperature.GetValue() {
				log.Info.Printf("temp: %dC\n", v.(int8))
				o.Temp.CurrentTemperature.SetValue(float64(v.(int8)))
			}
		case "SLI":
			// resp.Response is ID, resp.Parsed is name
			i, _ := strconv.ParseInt(string(resp.Response), 16, 32)
			if int(i) != o.Television.ActiveIdentifier.GetValue() {
				log.Info.Println("setting source from listener")
				o.Television.ActiveIdentifier.SetValue(int(i))
				o.Television.ConfiguredName.SetValue(fmt.Sprintf("%s:%s", a.Info.Name, o.Sources[int(i)]))
			}
		case "NRI":
			log.Info.Println("Onkyo Details pulled")
		case "NTM":
			// ignore
		case "NFI":
			// ignore
		case "NJA":
			// ignore
		case "UPD":
			log.Info.Printf("Update info: %s\n", resp.Parsed)
		case "NST":
			nps := v.(*eiscp.NetworkPlayStatus)
			switch nps.State {
			case "Play":
				if o.Television.CurrentMediaState.GetValue() != characteristic.CurrentMediaStatePlay {
					log.Info.Println("NST: Play")
					o.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePlay)
					// o.Television.Active.SetValue(characteristic.ActiveActive)
					// o.VolumeActive.SetValue(characteristic.ActiveActive)
				}
			case "Stop":
				if o.Television.CurrentMediaState.GetValue() != characteristic.CurrentMediaStateStop {
					o.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateStop)
					log.Info.Println("NST: Stop")
					// o.Television.Active.SetValue(characteristic.ActiveInactive)
					// o.VolumeActive.SetValue(characteristic.ActiveInactive)
				}
			case "Pause":
				if o.Television.CurrentMediaState.GetValue() != characteristic.CurrentMediaStatePause {
					o.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePause)
					log.Info.Println("NST: Pause")
					// o.Television.Active.SetValue(characteristic.ActiveInactive)
					// o.VolumeActive.SetValue(characteristic.ActiveInactive)
				}
			default:
				log.Info.Println("Unknown media state")
				o.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateUnknown)
				log.Info.Println("NST: unknown")
				// o.Television.Active.SetValue(characteristic.ActiveInactive)
				// o.VolumeActive.SetValue(characteristic.ActiveInactive)
			}
		case "MOT":
			log.Info.Printf("Music Optimizer: %t\n", v.(bool))
		case "DIM":
			log.Info.Printf("Dimmer: %s\n", v.(string))
		case "RAS":
			log.Info.Printf("Cinema Filter: %t\n", v.(bool))
		case "PCT":
			log.Info.Printf("Phase Control: %t\n", v.(bool))
		case "NDS":
			log.Info.Printf("Network: %+v\n", v.(*eiscp.NetworkStatus))
		default:
			log.Info.Printf("unhandled response on listener: %s %+v\n", resp.Command, v)
		}
	}
}
