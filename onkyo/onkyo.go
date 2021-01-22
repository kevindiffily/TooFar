package onkyo

import (
	// "encoding/json"
	"github.com/cloudkucooland/go-eiscp"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/platform"

	"fmt"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"strconv"
	"sync"
	"time"
)

// Platform is the handle to the OWM sensors
type Platform struct {
	Running bool
}

var onkyos map[string]*tfaccessory.TFAccessory
var doOnce sync.Once

// Startup is called by the platform management to get things going
func (o Platform) Startup(c *config.Config) platform.Control {
	o.Running = true
	return o
}

// Shutdown is called by the platform management to shut things down
func (o Platform) Shutdown() platform.Control {
	o.Running = false
	return o
}

// AddAccessory adds an Onkyo  device and registers it with HC
func (o Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	doOnce.Do(func() {
		onkyos = make(map[string]*tfaccessory.TFAccessory)
	})

	a.Type = accessory.TypeTelevision

	var err error
	dev, err := eiscp.NewReceiver(a.IP, true)
	if err != nil {
		log.Info.Printf(err.Error())
		return
	}
	deets, err := dev.GetDetails()
	if err != nil {
		log.Info.Printf("unable to pull for details: %s", err.Error())
		return
	}
	// j, _ := json.MarshalIndent(deets.Device.ZoneList, "", "  ")
	// log.Info.Printf("\n%s\n", j)

	a.Info.Manufacturer = deets.Device.Brand
	a.Info.Model = deets.Device.Model
	a.Info.SerialNumber = deets.Device.DeviceSerial
	a.Info.FirmwareRevision = deets.Device.FirmwareVersion
	a.Info.Name = fmt.Sprintf("%s (%s)", a.Name, deets.Device.ZoneList.Zone[0].Name)

	if a.Info.ID == 0 {
		s, err := strconv.ParseUint(deets.Device.DeviceSerial, 16, 64)
		if err != nil {
			log.Info.Println(err)
		}
		a.Info.ID = s
	}

	onkyos[a.Name] = a

	// add to HC for GUI
	log.Info.Printf("adding [%s]: [%s]", a.Info.Name, a.Info.Model)
	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)

	a.TXNR686.Amp = dev

	a.TXNR686.Television.ConfiguredName.SetValue(a.Info.Name)
	a.TXNR686.AddInputs(deets)
	a.TXNR686.AddZones(deets)

	go func(a *tfaccessory.TFAccessory) {
		iscpListener(a)
	}(a)

	// set initial power state
	power, err := a.TXNR686.Amp.GetPower()
	if err != nil {
		log.Info.Println(err.Error())
	}
	a.TXNR686.Television.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting power to %t", newstate)
		res, err := a.TXNR686.Amp.SetPower(newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
		log.Info.Printf("set power to: %+v", res.Response)
	})

	_, err = a.TXNR686.Amp.GetVolume()
	if err != nil {
		log.Info.Println(err.Error())
	}
	a.TXNR686.Volume.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("setting volume to: %d", newstate)
		vol, err := a.TXNR686.Amp.SetVolume(uint8(newstate))
		if err != nil {
			log.Info.Println(err.Error())
		}
		log.Info.Printf("set volume to: %d", vol)
	})

	a.TXNR686.Speaker.Mute.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting mute to: %t", newstate)
		mute, err := a.TXNR686.Amp.SetMute(newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
		log.Info.Printf("set mute to: %t", mute)
	})

	a.TXNR686.VolumeSelector.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("set volumeselector: %d", newstate)
	})

	_, err = a.TXNR686.Amp.GetTempData()
	if err != nil {
		log.Info.Println(err.Error())
	}
	a.TXNR686.Television.ActiveIdentifier.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("Setting input to %02X", newstate)
		source, err := a.TXNR686.Amp.SetSourceByCode(newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
		log.Info.Printf("set input to %+v", source.Response)
	})

	source, err := a.TXNR686.Amp.GetSource()
	if err != nil {
		log.Info.Println(err.Error())
	} else {
		i, _ := strconv.ParseInt(string(source), 16, 32)
		a.TXNR686.Television.ActiveIdentifier.SetValue(int(i))
		a.TXNR686.Television.ConfiguredName.SetValue(fmt.Sprintf("%s:%s", a.Info.Name, a.TXNR686.Sources[int(i)]))
	}

	/// NPS does not respond if powered off or noet set to network
	a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateUnknown)
	if power && source == eiscp.SrcNetwork {
		a.TXNR686.Amp.GetNetworkPlayStatus()
	}

	a.TXNR686.Television.RemoteKey.OnValueRemoteUpdate(func(newstate int) {
		d := a.Amp // not a.TXNR686.Amp ?
		switch newstate {
		case characteristic.RemoteKeyRewind:
			if err := d.SetOnly("NTC", "REW"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyFastForward:
			if err := d.SetOnly("NTC", "FF"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyExit:
			if err := d.SetOnly("NTC", "RETURN"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyPlayPause:
			if err := d.SetOnly("NTC", "P/P"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyInfo:
			if err := d.SetOnly("NTC", "TOP"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyNextTrack:
			if err := d.SetOnly("NTC", "TRUP"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyPrevTrack:
			if err := d.SetOnly("NTC", "TRDN"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyArrowUp:
			if err := d.SetOnly("NTC", "UP"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyArrowDown:
			if err := d.SetOnly("NTC", "DOWN"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyArrowLeft:
			if err := d.SetOnly("NTC", "LEFT"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyArrowRight:
			if err := d.SetOnly("NTC", "RIGHT"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeySelect:
			if err := d.SetOnly("NTC", "SELECT"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyBack:
			if err := d.SetOnly("NTC", "TOP"); err != nil {
				log.Info.Println(err)
			}
		}
	})

	a.Runner = runner
}

func iscpListener(a *tfaccessory.TFAccessory) {
	for resp := range a.TXNR686.Amp.Responses {
		v, err := resp.ParseResponseValue()
		if err != nil {
			log.Info.Println(err.Error())
			continue
		}
		switch resp.Command {
		case "PWR":
			if a.TXNR686.Television.On.GetValue() != v.(bool) {
				p := 0
				if v.(bool) {
					p = 1
				}
				a.TXNR686.Television.On.SetValue(v.(bool))
				a.TXNR686.Television.Active.SetValue(p)
				// a.TXNR686.VolumeActive.SetValue(p) // speaker
			}
		case "MVL":
			if int(v.(uint8)) != a.TXNR686.Television.Volume.GetValue() {
				a.TXNR686.Television.Volume.SetValue(int(v.(uint8)))
			}
		case "AMT":
			if v.(bool) != a.TXNR686.Speaker.Mute.GetValue() {
				a.TXNR686.Speaker.Mute.SetValue(v.(bool))
			}
		case "TPD":
			if float64(v.(int)) != a.TXNR686.Temp.CurrentTemperature.GetValue() {
				// log.Info.Println("setting temp from listener")
				a.TXNR686.Temp.CurrentTemperature.SetValue(float64(v.(int)))
			}
		case "SLI":
			i, _ := strconv.ParseInt(string(resp.Response), 16, 32)
			if int(i) != a.TXNR686.Television.ActiveIdentifier.GetValue() {
				log.Info.Println("setting source from listener")
				a.TXNR686.Television.ActiveIdentifier.SetValue(int(i))
				// a.TXNR686.Television.ConfiguredName.SetValue(fmt.Sprintf("%s:%s", a.Info.Name, a.TXNR686.Sources[int(i)]))
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
			log.Info.Printf("Update info: %s\n", resp.Response)
		case "NST":
			nps := v.(*eiscp.NetworkPlayStatus)
			switch nps.State {
			case "Play":
				if a.TXNR686.Television.CurrentMediaState.GetValue() != characteristic.CurrentMediaStatePlay {
					a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePlay)
					a.TXNR686.Television.Active.SetValue(characteristic.ActiveActive)
				}
			case "Stop":
				if a.TXNR686.Television.CurrentMediaState.GetValue() != characteristic.CurrentMediaStateStop {
					a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateStop)
					a.TXNR686.Television.Active.SetValue(characteristic.ActiveInactive)
				}
			case "Pause":
				if a.TXNR686.Television.CurrentMediaState.GetValue() != characteristic.CurrentMediaStatePause {
					a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePause)
					a.TXNR686.Television.Active.SetValue(characteristic.ActiveInactive)
				}
			default:
				log.Info.Println("Unknown media state")
				a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateUnknown)
				a.TXNR686.Television.Active.SetValue(characteristic.ActiveInactive)
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

func runner(a *tfaccessory.TFAccessory, d *action.Action) {
	// log.Info.Printf("in onkyo action runner: %+v", d)
	switch d.Verb {
	case "Stop":
		log.Info.Printf("called stop")
		a.Amp.SetNetworkPlayStatus("Sxx")
		a.TXNR686.Television.Active.SetValue(0)
		a.TXNR686.VolumeActive.SetValue(0)
	case "TuneInPreset":
		// http://vtochq-it.blogspot.com/2018/12/onkyo-pioneer-network-remote-control.html
		// log.Info.Printf("setting preset to %s", d.Value)
		if a.TXNR686.Television.On.GetValue() {
			a.TXNR686.Amp.SetPower(true)
			a.TXNR686.Television.On.SetValue(true)
			a.TXNR686.Television.Active.SetValue(1)
			// a.TXNR686.VolumeActive.SetValue(1)
			time.Sleep(time.Second)
		}

		source, err := a.TXNR686.Amp.GetSource()
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		if source != eiscp.SrcNetwork {
			_, err := a.TXNR686.Amp.SetSource(eiscp.SrcNetwork)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			time.Sleep(time.Second)
		}

		log.Info.Println("setting to tuneIN radio")
		err = a.TXNR686.Amp.SetNetworkServiceTuneIn()
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		time.Sleep(time.Second * 2)
		log.Info.Println("setting presets screen")
		err = a.TXNR686.Amp.SelectNetworkListItem(1) // first item in the list is "Presets"
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		time.Sleep(time.Second * 2)
		log.Info.Printf("setting to selected preset: %s", d.Value)
		pi, err := strconv.Atoi(d.Value)
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		err = a.TXNR686.Amp.SelectNetworkListItem(pi)
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
	default:
		log.Info.Printf("unknown verb %s (valid: TuneInPreset, Stop)", d.Verb)
	}
}

// GetAccessory looks up an onkyo device
func (o Platform) GetAccessory(name string) (*tfaccessory.TFAccessory, bool) {
	val, ok := onkyos[name]
	return val, ok
}

// Background starts up the go process to periodically update the onkyo values
func (o Platform) Background() {
	go func() {
		for range time.Tick(time.Minute * 1) {
			o.backgroundPuller()
		}
	}()
}

// we just ask, let the persistentListener process the responses
func (o Platform) backgroundPuller() {
	for _, a := range onkyos {
		power, err := a.TXNR686.Amp.GetPower()
		if err != nil {
			log.Info.Println(err.Error())
		}

		_, err = a.TXNR686.Amp.GetTempData()
		if err != nil {
			log.Info.Println(err.Error())
		}

		_, err = a.TXNR686.Amp.GetVolume()
		if err != nil {
			log.Info.Println(err.Error())
		}

		_, err = a.TXNR686.Amp.GetMute()
		if err != nil {
			log.Info.Println(err.Error())
		}

		source, err := a.TXNR686.Amp.GetSource()
		if err != nil {
			log.Info.Println(err.Error())
		}

		if power && source == eiscp.SrcNetwork {
			_, err := a.TXNR686.Amp.GetNetworkPlayStatus()
			if err != nil {
				log.Info.Println(err.Error())
			}
		}
	}
}
