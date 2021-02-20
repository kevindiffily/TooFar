package samsung

import (
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"

	"encoding/json"
	"github.com/brutella/hc/accessory"
	// "github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/util"
	"strconv"
	"sync"
	// "time"
	"encoding/hex"
	"github.com/McKael/samtv"
)

// Platform is the handle to the OWM sensors
type Platform struct {
	Running bool
}

var samsungs map[string]*tfaccessory.TFAccessory
var inputNames map[uint8]string
var inputKeys map[uint8]string
var storage util.Storage
var doOnce sync.Once

// Startup is called by the platform management to get things going
func (o Platform) Startup(c *config.Config) platform.Control {
	o.Running = true

	var err error
	storage, err = util.NewFileStorage("cache")
	if err != nil {
		log.Info.Printf(err.Error())
	}

	return o
}

// Shutdown is called by the platform management to shut things down
func (o Platform) Shutdown() platform.Control {
	o.Running = false
	return o
}

func (o Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	doOnce.Do(func() {
		samsungs = make(map[string]*tfaccessory.TFAccessory)
		inputNames = make(map[uint8]string)
		inputKeys = make(map[uint8]string)

		var inputs = []struct {
			ID   uint8
			Name string
			Key  string
		}{
			{1, "- HDMI3", "KEY_HDMI3"},
			{2, "- TV", "KEY_TV"},
			{3, "HDMI1", "KEY_HDMI1"},
			{4, "HDMI2", "KEY_HDMI2"},
			{5, "Favorite", "KEY_FAVCH"},
			{6, "Component 1", "KEY_COMPONENT1"},
			{7, "Component 2", "KEY_COMPONENT2"},
			{8, "Apps", "KEY_APP_LIST"},
		}

		for _, v := range inputs {
			inputNames[v.ID] = v.Name
			inputKeys[v.ID] = v.Key
		}

	})

	a.Type = accessory.TypeTelevision

	dev, err := samtv.NewSmartViewSession(a.IP)
	if err != nil {
		log.Info.Printf(err.Error())
		return
	}
	sessionKey, err := hex.DecodeString(a.Password)
	if err != nil {
		log.Info.Printf(err.Error())
		return
	}
	dev.RestoreSessionData(sessionKey, 1, a.Username)
	if err := dev.InitSession(); err != nil {
		log.Info.Printf(err.Error())
		return
	}

	deets, err := dev.DeviceDescription()
	if err != nil {
		log.Info.Printf(err.Error())
		return
	}

	a.Info.Manufacturer = "Samsung"

	log.Info.Printf("%+v", deets)
	// if we got nothing, don't panic, just read it from cache -- probably powered off
	if deets.ModelName == "" {
		raw, err := storage.Get(a.IP)
		if err != nil {
			log.Info.Println(err.Error())
		}

		err = json.Unmarshal(raw, &deets)
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
	} else {
		// if we got something, write it to the cache
		raw, err := json.Marshal(deets)
		if err != nil {
			log.Info.Println(err.Error())
		} else {
			log.Info.Println(string(raw))
			storage.Set(a.IP, raw)
		}
	}

	a.Info.Model = deets.ModelName
	a.Info.SerialNumber = deets.DeviceID
	a.Info.FirmwareRevision = deets.FirmwareVersion
	a.Info.Name = deets.DeviceName

	if a.Info.ID == 0 {
		s, err := strconv.ParseUint(deets.DUID[5:13], 16, 64)
		if err != nil {
			log.Info.Println(err.Error())
		}
		a.Info.ID = s
	}

	samsungs[a.Name] = a

	// add to HC for GUI
	log.Info.Printf("adding [%s]: [%s]", a.Info.Name, a.Info.Model)
	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)

	d := a.Device.(*devices.SamsungTV)
	d.SamTV = dev

	d.Television.ConfiguredName.SetValue(a.Info.Name)
	d.AddInputs(inputNames)

	d.Television.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting power to %t", newstate)
		if err := d.SamTV.Key("KEY_POWER"); err != nil {
			d.Television.On.SetValue(false)
			log.Info.Println(err.Error())
		}
	})

	d.Volume.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("volume not yet supported", newstate)
	})

	d.Television.ActiveIdentifier.OnValueRemoteUpdate(func(newstate int) {
		k, ok := inputKeys[uint8(newstate)]
		if !ok {
			k = "KEY_HDMI3"
		}
		log.Info.Printf("Setting input to %s", k)
		err := d.SamTV.Key(k)
		if err != nil {
			log.Info.Println(err.Error())
			d.Television.On.SetValue(false)
		}
	})

	d.Television.RemoteKey.OnValueRemoteUpdate(func(newstate int) {
		handleRemote(a, newstate)
	})

	a.Runner = runner
}

// GetAccessory looks up an onkyo device
func (o Platform) GetAccessory(name string) (*tfaccessory.TFAccessory, bool) {
	val, ok := samsungs[name]
	return val, ok
}

// Background starts up the go process to periodically update the onkyo values
func (o Platform) Background() {
	/* go func() {
		for range time.Tick(time.Minute * 1) {
			o.backgroundPuller()
		}
	}() */
}

func (o Platform) backgroundPuller() {
	// gotta figure out how to get something useful out of the TV first...
}
