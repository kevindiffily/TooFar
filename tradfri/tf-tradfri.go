package tradfri

import (
	"github.com/brutella/hc/log"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/platform"

	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/brutella/hc/accessory"
	"github.com/eriklupander/dtls"
	"github.com/eriklupander/tradfri-go/model"
	"github.com/sirupsen/logrus"

	"github.com/ghetzel/go-stockutil/colorutil"
)

// Platform is the handle to the bridge
type Platform struct {
	Running bool
}

// DevicePlatform is the handle to the devices on the bridge
type DevicePlatform struct {
	Running bool
}

// tradfris are the various bridges
var tradfris map[string]*tfaccessory.TFAccessory

// tradfriDevices are the devices on the bridges
var tradfriDevices map[string]*tfaccessory.TFAccessory
var tradfriClient *Client

var doOnceTradfri sync.Once

// Startup is called by the platform management to start things up
func (tp Platform) Startup(c *config.Config) platform.Control {
	tp.Running = true
	return tp
}

// Shutdown is called by the platform management to shut things down
func (tp Platform) Shutdown() platform.Control {
	tp.Running = false
	return tp
}

// AddAccessory is called to add a Tradfri bridge/gateway -- devices are enumerated automatically from it
func (tp Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	doOnceTradfri.Do(func() {
		tradfris = make(map[string]*tfaccessory.TFAccessory)
		var tdp DevicePlatform
		platform.RegisterPlatform("Tradfri-Device", tdp)
		tradfriDevices = make(map[string]*tfaccessory.TFAccessory)
		dtls.SetLogLevel(dtls.LogLevelError)
		logrus.SetLevel(logrus.ErrorLevel)
	})
	h, _ := platform.GetPlatform("HomeControl")

	// add the gateway to our list
	tradfris[a.IP] = a
	log.Info.Printf("Adding Tradfri Gateway: [%s]", a.IP)
	IP := fmt.Sprintf("%s:%d", a.IP, 5684)

	tradfriClient = NewTradfriClient(IP, a.Username, a.Password)
	devs, err := tradfriClient.ListDevices()
	if err != nil {
		log.Info.Printf(err.Error())
		return
	}

	for _, d := range devs {
		did := fmt.Sprintf("%d", d.DeviceId)
		newDevice := tfaccessory.TFAccessory{
			Platform: "Tradfri-Device",
			Name:     did,
			Info: accessory.Info{
				Name:             d.Name,
				SerialNumber:     d.Metadata.SerialNumber,
				Manufacturer:     d.Metadata.Vendor,
				Model:            d.Metadata.TypeName,
				FirmwareRevision: d.Metadata.TypeId,
				ID:               uint64(d.DeviceId),
			},
		}
		if newDevice.Info.SerialNumber == "" {
			newDevice.Info.SerialNumber = d.Name
		}
		log.Info.Printf("Adding: [%s]: [%s]", newDevice.Info.Name, newDevice.Info.Model)

		switch d.Type {
		case DeviceTypeRemote:
			newDevice.Type = accessory.TypeRemoteControl
			// h.AddAccessory // -- unsupported by HomeKit
		case DeviceTypeSlaveRemote:
			newDevice.Type = accessory.TypeRemoteControl
			// h.AddAccessory // -- unsupported by HomeKit
		case DeviceTypeLightbulb:
			newDevice.Type = accessory.TypeLightbulb
			h.AddAccessory(&newDevice)
			lightbulbLogic(&newDevice, d)
		case DeviceTypePlug:
			newDevice.Type = accessory.TypeOutlet
			newDevice.Switch.Switch.On.OnValueRemoteUpdate(func(newstate bool) {
				log.Info.Printf("setting [%s] to [%t] from Tradfri-Device lightbulb handler", newDevice.Name, newstate)
				_, err := tradfriClient.PutDevicePower(newDevice.Name, newstate)
				if err != nil {
					log.Info.Println(err.Error())
				}
			})
			// h.AddAccessory
		case DeviceTypeMotionSensor:
			newDevice.Type = accessory.TypeSensor
			// h.AddAccessory
		case DeviceTypeSignalRepeater:
			newDevice.Type = accessory.TypeOther
			// h.AddAccessory // -- unsupported by HomeKit
		case DeviceTypeBlind:
			newDevice.Type = accessory.TypeWindowCovering
			// h.AddAccessory
		case DeviceTypeSoundRemote:
			newDevice.Type = accessory.TypeRemoteControl
			// h.AddAccessory // -- unsupported by HomeKit
		}

		newDevice.Runner = runner
		tradfriDevices[did] = &newDevice
	}
}

func lightbulbLogic(newDevice *tfaccessory.TFAccessory, d model.Device) {
	switch newDevice.Info.Model {
	case "TRADFRI bulb E26 CWS opal 600lm":
		lightbulbHSL(newDevice, d)
	case "TRADFRI bulb E26 WS opal 980lm":
		lightbulbTemp(newDevice, d)
	case "LTD010":
		lightbulbTemp(newDevice, d)
	default:
		log.Info.Printf("unknown bulb type, using generic: %s", newDevice.Info.Model)
		lightbulbSimple(newDevice, d)
	}
}

func lightbulbSimple(newDevice *tfaccessory.TFAccessory, d model.Device) {
	newDevice.Lightbulb.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)
	// handlers
	newDevice.Lightbulb.Lightbulb.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("Tradfri-Device Simple Lightbulb handler setting [%s] to [%t]", newDevice.Name, newstate)
		_, err := tradfriClient.PutDevicePower(newDevice.Name, newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
}

func lightbulbTemp(newDevice *tfaccessory.TFAccessory, d model.Device) {
	newDevice.TempLightbulb.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)

	newDevice.TempLightbulb.Lightbulb.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("Tradfri-Device Temp Lightbulb handler setting [%s] to [%t]", newDevice.Name, newstate)
		_, err := tradfriClient.PutDevicePower(newDevice.Name, newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})

	dv := int(mapRange(float64(d.LightControl[0].Dimmer), 0, 254, 0, 100))
	newDevice.TempLightbulb.Lightbulb.Brightness.SetValue(dv)
	newDevice.TempLightbulb.Lightbulb.Brightness.OnValueRemoteUpdate(func(newval int) {
		log.Info.Printf("Tradfri-Device Temp Lightbulb handler setting [%s] brightness: %d", newDevice.Name, newval)
		val := newval * 255 / 100
		_, err := tradfriClient.PutDeviceDimming(newDevice.Name, val)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
	newDevice.TempLightbulb.Lightbulb.ColorTemperature.SetValue(dv)
	newDevice.TempLightbulb.Lightbulb.ColorTemperature.OnValueRemoteUpdate(func(newval int) {
		// this is right at the extremes and wrong in the middle, which is fine with me
		kelvin := int(mapRange(float64(newval), float64(140), float64(500), float64(7142), float64(2000)))
		log.Info.Printf("Tradfri-Device Temp Lightbulb handler setting [%s] temperature: %d (%d K) ", newDevice.Name, newval, kelvin)
		color := colorutil.KelvinToColor(kelvin)
		h, s, v := color.HSV() // 360, 1, 1 -- max values
		ds := s * 100          // normalize 1-100
		dv := v * 100          // normalize 1-100
		_, err := tradfriClient.PutDeviceColorHSL(newDevice.Name, h, ds, dv)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
}

func lightbulbHSL(newDevice *tfaccessory.TFAccessory, d model.Device) {
	newDevice.ColoredLightbulb.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)
	dv := int(mapRange(float64(d.LightControl[0].Dimmer), 0, 254, 0, 100))
	newDevice.ColoredLightbulb.Lightbulb.Brightness.SetValue(dv)
	dhue := mapRange(float64(d.LightControl[0].Hue), 0, 65279, 0, 360)
	newDevice.ColoredLightbulb.Lightbulb.Hue.SetValue(dhue)
	dsat := mapRange(float64(d.LightControl[0].Saturation), 0, 65279, 0, 100)
	newDevice.ColoredLightbulb.Lightbulb.Saturation.SetValue(dsat)

	// handlers
	newDevice.ColoredLightbulb.Lightbulb.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("Tradfri-Device HSL handler setting [%s] to [%t]", newDevice.Name, newstate)
		_, err := tradfriClient.PutDevicePower(newDevice.Name, newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
	newDevice.ColoredLightbulb.Lightbulb.Brightness.OnValueRemoteUpdate(func(newval int) {
		log.Info.Printf("Tradfri-Device HSL handler setting [%s] brightness: %d", newDevice.Name, newval)
		val := newval * 255 / 100
		_, err := tradfriClient.PutDeviceDimming(newDevice.Name, val)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
	newDevice.ColoredLightbulb.Lightbulb.Hue.OnValueRemoteUpdate(func(newval float64) {
		log.Info.Printf("Tradfri-Device HSL handler setting [%s] hue: %f", newDevice.Name, newval)
		dev, err := tradfriClient.GetDevice(newDevice.Name)
		if err != nil {
			log.Info.Printf(err.Error())
			return
		}
		// HC sends 0-360
		// convert it so it can be converted back...
		curS := mapRange(float64(dev.LightControl[0].Saturation), 0, 65279, 0, 100)
		curL := mapRange(float64(dev.LightControl[0].Dimmer), 0, 254, 0, 100)
		_, err = tradfriClient.PutDeviceColorHSL(newDevice.Name, newval, curS, curL)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
	newDevice.ColoredLightbulb.Lightbulb.Saturation.OnValueRemoteUpdate(func(newval float64) {
		log.Info.Printf("Tradfri-Device HSL handler setting [%s] saturation: %f", newDevice.Name, newval)
		dev, err := tradfriClient.GetDevice(newDevice.Name)
		if err != nil {
			log.Info.Printf(err.Error())
			return
		}
		// HC sends 0-100
		// convert it so it can be converted back...
		curH := mapRange(float64(dev.LightControl[0].Hue), 0, 65279, 0, 360)
		curL := mapRange(float64(dev.LightControl[0].Dimmer), 0, 254, 0, 100)
		_, err = tradfriClient.PutDeviceColorHSL(newDevice.Name, curH, newval, curL)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
}

// this is not full-featured, but meets my limited needs
func runner(a *tfaccessory.TFAccessory, action *action.Action) {
	log.Info.Printf("in tradfri-device action runner: %+v %+v", a, action)
	switch action.Verb {
	case "SetBrightness":
		target, _ := strconv.Atoi(action.Value)

		// if it is already at the target, set to full-on
		bv := a.ColoredLightbulb.Lightbulb.Brightness.GetValue()
		log.Info.Printf("current brightness: %d, target: %d", bv, target)
		if bv == action.SetTargetBrightness {
			log.Info.Printf("adjusting target too 99")
			target = 99
		}

		// update hardware
		_, err := tradfriClient.PutDeviceDimming(a.Name, target*255/100)
		if err != nil {
			log.Info.Println(err.Error())
		}

		// update GUI
		a.ColoredLightbulb.Lightbulb.Brightness.SetValue(target)
	case "Toggle":
		log.Info.Println("toggle verb called")
	default:
		log.Info.Println("unknown tradfri verb: %s", action.Verb)
	}

	if action.SetTargetBrightness != 0 {
		log.Info.Println("switch to using SetBrightness verb")
		target := action.SetTargetBrightness

		// if it is already at the target, set to full-on
		if a.ColoredLightbulb != nil {
			bv := a.ColoredLightbulb.Lightbulb.Brightness.GetValue()
			log.Info.Printf("current brightness: %d, target: %d", bv, target)
			if bv == action.SetTargetBrightness {
				log.Info.Printf("adjusting target too 99")
				target = 99
			}

			// update hardware
			_, err := tradfriClient.PutDeviceDimming(a.Name, target*255/100)
			if err != nil {
				log.Info.Println(err.Error())
			}

			// update GUI
			a.ColoredLightbulb.Lightbulb.Brightness.SetValue(target)
		}

		if a.TempLightbulb != nil {
			bv := a.TempLightbulb.Lightbulb.Brightness.GetValue()
			log.Info.Printf("current brightness: %d, target: %d", bv, target)
			if bv == action.SetTargetBrightness {
				log.Info.Printf("adjusting target too 99")
				target = 99
			}

			// update hardware
			_, err := tradfriClient.PutDeviceDimming(a.Name, target*255/100)
			if err != nil {
				log.Info.Println(err.Error())
			}

			// update GUI
			a.TempLightbulb.Lightbulb.Brightness.SetValue(target)
		}

	} else {
		log.Info.Println("switch to using SetBrightness verb")
		// just toggle
		newstate := !a.ColoredLightbulb.Lightbulb.On.GetValue()
		_, err := tradfriClient.PutDevicePower(a.Name, newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
		a.ColoredLightbulb.Lightbulb.On.SetValue(newstate)
	}
}

// GetAccessory gets the bridge by IP address
func (tp Platform) GetAccessory(ip string) (*tfaccessory.TFAccessory, bool) {
	val, ok := tradfris[ip]
	return val, ok
}

// Background runs the background tasks for the bridges (none)
func (tp Platform) Background() {
	// nothing
}

// AddAccessory is called to add individual accessories -- do not use directly
func (tdp DevicePlatform) AddAccessory(a *tfaccessory.TFAccessory) {
	log.Info.Println("do not add tradfri devices, add the gateway and the devices are auto-added")
}

// GetAccessory returns a Tradfri Device accessory by name
func (tdp DevicePlatform) GetAccessory(name string) (*tfaccessory.TFAccessory, bool) {
	tdd, ok := tradfriDevices[name]
	return tdd, ok
}

// Startup the TradfriDevicePlatform -- never actually called
func (tdp DevicePlatform) Startup(c *config.Config) platform.Control {
	tdp.Running = true
	return tdp
}

// Shutdown the TradfriDevicePlatform -- never actually called
func (tdp DevicePlatform) Shutdown() platform.Control {
	return tdp
}

// Background runs the TradfriDevice status pollers
func (tdp DevicePlatform) Background() {
	go func() {
		for range time.Tick(time.Minute) {
			tradfriUpdateAll()
		}
	}()
}

func tradfriUpdateAll() {
	// log.Info.Println("running Tradfri-Device background tasks")
	td, ok := platform.GetPlatform("Tradfri-Device")
	if !ok {
		log.Info.Printf("unable to get Tradfri-Device platform")
		return
	}

	devs, err := tradfriClient.ListDevices()
	if err != nil {
		log.Info.Printf(err.Error())
		return
	}

	for _, d := range devs {
		did := fmt.Sprintf("%d", d.DeviceId)
		tdd, ok := td.GetAccessory(did)
		if !ok {
			log.Info.Printf("unable to get Tradfri-Device [%s]", did)
			continue
		}

		switch d.Type {
		case DeviceTypeLightbulb:
			if tdd.ColoredLightbulb != nil {
				if tdd.ColoredLightbulb.Lightbulb.On.GetValue() != (d.LightControl[0].Power > 0) {
					tdd.ColoredLightbulb.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)
				}
				dv := int(mapRange(float64(d.LightControl[0].Dimmer), 0, 254, 0, 100))
				tdd.ColoredLightbulb.Lightbulb.Brightness.SetValue(dv)
				dhue := mapRange(float64(d.LightControl[0].Hue), 0, 65279, 0, 360)
				tdd.ColoredLightbulb.Lightbulb.Hue.SetValue(dhue)
				dsat := mapRange(float64(d.LightControl[0].Saturation), 0, 65279, 0, 100)
				tdd.ColoredLightbulb.Lightbulb.Saturation.SetValue(dsat)
			}
			if tdd.Lightbulb != nil {
				if tdd.Lightbulb.Lightbulb.On.GetValue() != (d.LightControl[0].Power > 0) {
					tdd.Lightbulb.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)
				}
			}
			if tdd.TempLightbulb != nil {
				if tdd.TempLightbulb.Lightbulb.On.GetValue() != (d.LightControl[0].Power > 0) {
					tdd.TempLightbulb.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)
				}
				dv := int(mapRange(float64(d.LightControl[0].Dimmer), 0, 254, 0, 100))
				if tdd.TempLightbulb.Lightbulb.Brightness.GetValue() != dv {
					tdd.TempLightbulb.Lightbulb.Brightness.SetValue(dv)
					// if you change the color in Ikea's app, ... well ... I can't convert from HSL to Kelvin yet
				}
			}
		case DeviceTypePlug:
			// this is wrong -- don't have a plug to test, just wanted something here for the switch to make sense
			// tdd.ColoredLightbulb.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)
		}
	}
}
