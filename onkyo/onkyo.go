package onkyo

import (
	"github.com/cloudkucooland/go-eiscp"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
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
	// we don't ever care about cover art, and can make the first pull fail
	dev.SetNetworkJacketArt(false)
	deets, err := dev.GetDetails()
	if err != nil {
		log.Info.Printf("unable to pull for details: %s", err.Error())
		return
	}

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

	d := a.Device.(*devices.TXNR686)
	d.Amp = dev

	d.Television.ConfiguredName.SetValue(a.Info.Name)
	d.AddInputs(deets)
	d.AddZones(deets)

	go func(a *tfaccessory.TFAccessory) {
		iscpListener(a)
	}(a)

	// set initial power state
	power, err := d.Amp.GetPower()
	if err != nil {
		log.Info.Println(err.Error())
	}
	d.Television.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting power to %t", newstate)
		_, err := d.Amp.SetPower(newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})

	_, err = d.Amp.GetVolume()
	if err != nil {
		log.Info.Println(err.Error())
	}
	d.Volume.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("setting volume to: %d", newstate)
		_, err := d.Amp.SetVolume(uint8(newstate))
		if err != nil {
			log.Info.Println(err.Error())
		}
	})

	d.Speaker.Mute.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting mute to: %t", newstate)
		_, err := d.Amp.SetMute(newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})

	d.VolumeSelector.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("set volumeselector: %d", newstate)
	})

	if _, err := d.Amp.GetTempData(); err != nil {
		log.Info.Println(err.Error())
	}

	d.Television.ActiveIdentifier.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("Setting input to %02X", newstate)
		_, err := d.Amp.SetSourceByCode(newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
	source, err := d.Amp.GetSource()
	if err != nil {
		log.Info.Println(err.Error())
	} else {
		i, _ := strconv.ParseInt(string(source), 16, 32)
		d.Television.ActiveIdentifier.SetValue(int(i))
		d.Television.ConfiguredName.SetValue(fmt.Sprintf("%s:%s", a.Info.Name, d.Sources[int(i)]))
	}

	/// NPS does not respond if powered off or not set to SLI network
	d.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateUnknown)
	if power && source == eiscp.SrcNetwork {
		d.Amp.GetNetworkPlayStatus()
	}

	d.Television.RemoteKey.OnValueRemoteUpdate(func(newstate int) {
		handleRemote(a, newstate)
	})

	a.Runner = runner
	addController(a)
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
		d := a.Device.(*devices.TXNR686)
		d.Amp.GetTempData()
		d.Amp.GetVolume()
		d.Amp.GetMute()

		power, err := d.Amp.GetPower()
		if err != nil {
			log.Info.Println(err.Error())
			err = nil
		}

		source, err := d.Amp.GetSource()
		if err != nil {
			log.Info.Println(err.Error())
			err = nil
		}

		if power && source == eiscp.SrcNetwork {
			if _, err := d.Amp.GetNetworkPlayStatus(); err != nil {
				log.Info.Println(err.Error())
			}
		}
	}
}
