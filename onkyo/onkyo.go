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

	dev, err := eiscp.NewReceiver(a.IP)
	if err != nil {
		log.Info.Printf(err.Error())
		return
	}
	defer dev.Close()
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

	a.TXNR686.Television.ConfiguredName.SetValue(a.Info.Name)
	a.TXNR686.AddInputs(deets)
	a.TXNR686.AddZones(deets)

	// set initial power state
	power, err := dev.GetPower()
	if err != nil {
		log.Info.Println(err.Error())
	} else {
		p := 0
		if power {
			p = 1
		}
		a.TXNR686.Television.Active.SetValue(p)
		a.TXNR686.VolumeActive.SetValue(p) // speaker
		a.TXNR686.Television.On.SetValue(power)
	}
	a.TXNR686.Television.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting power to %t", newstate)
		dev.Connect()
		defer dev.Close()
		res, err := dev.SetPower(newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
		log.Info.Printf("set power to: %+v", res.Response)
	})

	// set initial volume
	vol, err := dev.GetVolume()
	if err != nil {
		log.Info.Println(err.Error())
	} else {
		log.Info.Printf("setting initial volume to: %d", vol)
		a.TXNR686.Volume.SetValue(int(vol))
	}
	a.TXNR686.Volume.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("setting volume to: %d", newstate)
		dev.Connect()
		defer dev.Close()
		vol, err := dev.SetVolume(uint8(newstate))
		if err != nil {
			log.Info.Println(err.Error())
		}
		log.Info.Printf("set volume to: %d", vol)
	})

	mute, err := dev.GetMute()
	if err != nil {
		log.Info.Println(err.Error())
	} else {
		log.Info.Printf("setting initial mute to: %t", mute)
		a.TXNR686.Speaker.Mute.SetValue(mute)
	}
	a.TXNR686.Speaker.Mute.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting mute to: %t", newstate)
		dev.Connect()
		defer dev.Close()
		mute, err := dev.SetMute(newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
		log.Info.Printf("set mute to: %t", mute)
	})

	a.TXNR686.VolumeSelector.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("set volumeselector: %d", newstate)
	})

	// set initial temp data
	cent, err := dev.GetTempData()
	if err != nil {
		log.Info.Println(err.Error())
	}
	cint, err := strconv.Atoi(cent)
	if err != nil {
		cint = 20
	}
	a.TXNR686.Temp.CurrentTemperature.SetValue(float64(cint))

	a.TXNR686.Television.ActiveIdentifier.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("Setting input to %02X", newstate)
		dev.Connect()
		defer dev.Close()
		source, err := dev.SetSourceByCode(newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
		log.Info.Printf("set input to %+v", source.Response)
	})

	source, err := dev.GetSource()
	if err != nil {
		log.Info.Println(err.Error())
	} else {
		i, _ := strconv.ParseInt(string(source), 16, 32)
		a.TXNR686.Television.ActiveIdentifier.SetValue(int(i))
		a.TXNR686.Television.ConfiguredName.SetValue(fmt.Sprintf("%s:%s", a.Info.Name, a.TXNR686.Sources[int(i)]))
	}

	/// NPS does not respond if powered off
	if power && source == eiscp.SrcNetwork {
		nps, err := dev.GetNetworkPlayStatus()
		log.Info.Printf("setting CurrentMediaState to: %+v", nps)
		if err != nil && nps != nil {
			switch nps.State {
			case "Play":
				a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePlay)
				a.TXNR686.Television.TargetMediaState.SetValue(characteristic.TargetMediaStatePlay)
			case "Stop":
				a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateStop)
				a.TXNR686.Television.TargetMediaState.SetValue(characteristic.TargetMediaStateStop)
			case "Pause":
				a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePause)
				a.TXNR686.Television.TargetMediaState.SetValue(characteristic.TargetMediaStatePause)
			default:
				a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateUnknown)
			}
		}
	}

	a.TXNR686.Television.RemoteKey.OnValueRemoteUpdate(func(newstate int) {
		d, err := eiscp.NewReceiver(a.IP)
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		defer d.Close()
		switch newstate {
		case characteristic.RemoteKeyRewind:
			log.Info.Println("TXNR686: RemoteKey: Rew")
			if err := d.SetOnly("NTC", "REW"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyFastForward:
			log.Info.Println("TXNR686: RemoteKey: FF")
			if err := d.SetOnly("NTC", "FF"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyExit:
			log.Info.Println("TXNR686: RemoteKey: Exit")
			if err := d.SetOnly("NTC", "RETURN"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyPlayPause:
			log.Info.Println("TXNR686: RemoteKey: PlayPause")
			if err := d.SetOnly("NTC", "P/P"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyInfo:
			log.Info.Println("TXNR686: RemoteKey: Info")
			if err := d.SetOnly("NTC", "TOP"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyNextTrack:
			log.Info.Println("TXNR686: RemoteKey: Next Track")
			if err := d.SetOnly("NTC", "TRUP"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyPrevTrack:
			log.Info.Println("TXNR686: RemoteKey: Prev Track")
			if err := d.SetOnly("NTC", "TRDN"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyArrowUp:
			log.Info.Println("TXNR686: RemoteKey: Arrow Up")
			if err := d.SetOnly("NTC", "UP"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyArrowDown:
			log.Info.Println("TXNR686: RemoteKey: Arrow Down")
			if err := d.SetOnly("NTC", "DOWN"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyArrowLeft:
			log.Info.Println("TXNR686: RemoteKey: Arrow Left")
			if err := d.SetOnly("NTC", "LEFT"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyArrowRight:
			log.Info.Println("TXNR686: RemoteKey: Arrow Right")
			if err := d.SetOnly("NTC", "RIGHT"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeySelect:
			log.Info.Println("TXNR686: RemoteKey: Select")
			if err := d.SetOnly("NTC", "SELECT"); err != nil {
				log.Info.Println(err)
			}
		case characteristic.RemoteKeyBack:
			log.Info.Println("TXNR686: RemoteKey: Back")
			if err := d.SetOnly("NTC", "TOP"); err != nil {
				log.Info.Println(err)
			}
		}
	})

	a.Runner = runner
}

func runner(a *tfaccessory.TFAccessory, d *action.Action) {
	// log.Info.Printf("in onkyo action runner: %+v", d)
	dev, err := eiscp.NewReceiver(a.IP)
	if err != nil {
		log.Info.Printf(err.Error())
		return
	}
	defer dev.Close()
	switch d.Verb {
	case "Stop":
		log.Info.Printf("called stop")
		dev.SetNetworkPlayStatus("Sxx")
		a.TXNR686.Television.Active.SetValue(0)
		a.TXNR686.VolumeActive.SetValue(0)
	case "TuneInPreset":
		// http://vtochq-it.blogspot.com/2018/12/onkyo-pioneer-network-remote-control.html
		// log.Info.Printf("setting preset to %s", d.Value)
		if a.TXNR686.Television.On.GetValue() {
			dev.SetPower(true)
			a.TXNR686.Television.On.SetValue(true)
			a.TXNR686.Television.Active.SetValue(1)
			a.TXNR686.VolumeActive.SetValue(1)
			time.Sleep(time.Second)
		}

		source, err := dev.GetSource()
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		if source != eiscp.SrcNetwork {
			_, err := dev.SetSource(eiscp.SrcNetwork)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			time.Sleep(time.Second)
		}

		i, _ := strconv.ParseInt(string(source), 16, 32)
		a.TXNR686.Television.ActiveIdentifier.SetValue(int(i))
		a.TXNR686.Television.ConfiguredName.SetValue(fmt.Sprintf("%s:%s", a.Info.Name, a.TXNR686.Sources[int(i)]))
		log.Info.Println("setting to tuneIN radio")
		dev.SetNetworkServiceTuneIn()
		time.Sleep(time.Second * 3)
		log.Info.Println("setting presets screen")
		dev.SelectNetworkListItem(1) // first item in the list is "Presets"
		time.Sleep(time.Second * 3)
		log.Info.Printf("setting to selected preset: %s", d.Value)
		pi, _ := strconv.Atoi(d.Value)
		dev.SelectNetworkListItem(pi)
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

func (o Platform) backgroundPuller() {
	for _, a := range onkyos {
		dev, err := eiscp.NewReceiver(a.IP)
		if err != nil {
			log.Info.Printf(err.Error())
			return
		}
		defer dev.Close()

		power, err := dev.GetPower()
		if err != nil {
			log.Info.Println(err.Error())
		} else {
			p := 0
			if power {
				p = 1
			}
			if power != a.TXNR686.Television.On.GetValue() {
				a.TXNR686.Television.Active.SetValue(p)
				a.TXNR686.VolumeActive.SetValue(p)
				a.TXNR686.Television.On.SetValue(power)
			}
		}

		cent, err := dev.GetTempData()
		if err != nil {
			log.Info.Println(err.Error())
		} else {
			cint, err := strconv.Atoi(cent)
			if err != nil {
				cint = 20
			}
			a.TXNR686.Temp.CurrentTemperature.SetValue(float64(cint))
		}

		vol, err := dev.GetVolume()
		if err != nil {
			log.Info.Println(err.Error())
		} else {
			if int(vol) != a.TXNR686.Television.Volume.GetValue() {
				a.TXNR686.Television.Volume.SetValue(int(vol))
			}
		}

		mute, err := dev.GetMute()
		if err != nil {
			log.Info.Println(err.Error())
		} else {
			if mute != a.TXNR686.Speaker.Mute.GetValue() {
				a.TXNR686.Speaker.Mute.SetValue(mute)
			}
		}

		source, err := dev.GetSource()
		if err != nil {
			log.Info.Println(err.Error())
		} else {
			i, _ := strconv.ParseInt(string(source), 16, 32)
			if int(i) != a.TXNR686.Television.ActiveIdentifier.GetValue() {
				a.TXNR686.Television.ActiveIdentifier.SetValue(int(i))
				a.TXNR686.Television.ConfiguredName.SetValue(fmt.Sprintf("%s:%s", a.Info.Name, a.TXNR686.Sources[int(i)]))
			}
		}

		if power && source == eiscp.SrcNetwork {
			nps, err := dev.GetNetworkPlayStatus()
			// log.Info.Printf("setting CurrentMediaState to: %+v", nps)
			if err != nil && nps != nil {
				switch nps.State {
				case "Play":
					if a.TXNR686.Television.CurrentMediaState.GetValue() != characteristic.CurrentMediaStatePlay {
						a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePlay)
						a.TXNR686.Television.TargetMediaState.SetValue(characteristic.TargetMediaStatePlay)
					}
				case "Stop":
					if a.TXNR686.Television.CurrentMediaState.GetValue() != characteristic.CurrentMediaStateStop {
						a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateStop)
						a.TXNR686.Television.TargetMediaState.SetValue(characteristic.TargetMediaStateStop)
					}
				case "Pause":
					if a.TXNR686.Television.CurrentMediaState.GetValue() != characteristic.CurrentMediaStatePause {
						a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePause)
						a.TXNR686.Television.TargetMediaState.SetValue(characteristic.TargetMediaStatePause)
					}
				default:
					a.TXNR686.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateUnknown)
				}
			}
		}
	}
}
