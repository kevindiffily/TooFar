package tradfri

// https://callistaenterprise.se/blogg/teknik/2019/03/15/a-quick-home-automation/

import (
	"github.com/brutella/hc/log"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"

	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/brutella/hc/accessory"
	"github.com/eriklupander/dtls"
	"github.com/eriklupander/tradfri-go/model"
	"github.com/sirupsen/logrus"
	// "github.com/ghetzel/go-stockutil/colorutil"
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
	case "LTD010": // philips hue can lightbulbs
		lightbulbTemp(newDevice, d)
	default:
		log.Info.Printf("unknown bulb type, using generic: %s", newDevice.Info.Model)
		lightbulbSimple(newDevice, d)
	}
}

func lightbulbSimple(newDevice *tfaccessory.TFAccessory, d model.Device) {
	lb := newDevice.Device.(*accessory.Lightbulb)
	lb.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)
	// handlers
	lb.Lightbulb.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("Tradfri-Device Simple Lightbulb handler setting [%s] to [%t]", newDevice.Name, newstate)
		_, err := tradfriClient.PutDevicePower(newDevice.Name, newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
}

func lightbulbTemp(newDevice *tfaccessory.TFAccessory, d model.Device) {
	tlb := newDevice.Device.(*devices.TempLightbulb)
	tlb.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)

	tlb.Lightbulb.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("Tradfri-Device Temp Lightbulb handler setting [%s] to [%t]", newDevice.Name, newstate)
		_, err := tradfriClient.PutDevicePower(newDevice.Name, newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})

	dv := int(mapRange(float64(d.LightControl[0].Dimmer), 0, 254, 0, 100))
	tlb.Lightbulb.Brightness.SetValue(dv)
	tlb.Lightbulb.Brightness.OnValueRemoteUpdate(func(newval int) {
		log.Info.Printf("Tradfri-Device Temp Lightbulb handler setting [%s] brightness: %d", newDevice.Name, newval)
		val := newval * 255 / 100
		_, err := tradfriClient.PutDeviceDimming(newDevice.Name, val)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
	tlb.Lightbulb.ColorTemperature.SetValue(dv)
	tlb.Lightbulb.ColorTemperature.OnValueRemoteUpdate(func(newval int) {
		// this is right at the extremes and wrong in the middle, which is fine with me
		kelvin := int(mapRange(float64(newval), float64(140), float64(500), float64(7142), float64(2000)))
		log.Info.Printf("Tradfri-Device Temp Lightbulb handler setting [%s] temperature: %d (%d K) ", newDevice.Name, newval, kelvin)
		/* color := colorutil.KelvinToColor(kelvin)
		 h, s, v := color.HSV() // 360, 1, 1 -- max values
		ds := s * 100          // normalize 1-100
		dv := v * 100          // normalize 1-100
		*/

		r, g, b := kelvinToRGB(kelvin)
		h, ds, dv := rgbToHsl(int(r), int(g), int(b))
		_, err := tradfriClient.PutDeviceColorHSL(newDevice.Name, h, ds, dv)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
}

func lightbulbHSL(newDevice *tfaccessory.TFAccessory, d model.Device) {
	hsl := newDevice.Device.(*accessory.ColoredLightbulb)

	hsl.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)
	dv := int(mapRange(float64(d.LightControl[0].Dimmer), 0, 254, 0, 100))
	hsl.Lightbulb.Brightness.SetValue(dv)
	dhue := mapRange(float64(d.LightControl[0].Hue), 0, 65279, 0, 360)
	hsl.Lightbulb.Hue.SetValue(dhue)
	dsat := mapRange(float64(d.LightControl[0].Saturation), 0, 65279, 0, 100)
	hsl.Lightbulb.Saturation.SetValue(dsat)

	// handlers
	hsl.Lightbulb.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("Tradfri-Device HSL handler setting [%s] to [%t]", newDevice.Name, newstate)
		_, err := tradfriClient.PutDevicePower(newDevice.Name, newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
	hsl.Lightbulb.Brightness.OnValueRemoteUpdate(func(newval int) {
		log.Info.Printf("Tradfri-Device HSL handler setting [%s] brightness: %d", newDevice.Name, newval)
		val := newval * 255 / 100
		_, err := tradfriClient.PutDeviceDimming(newDevice.Name, val)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})
	hsl.Lightbulb.Hue.OnValueRemoteUpdate(func(newval float64) {
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
	hsl.Lightbulb.Saturation.OnValueRemoteUpdate(func(newval float64) {
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
		bv := a.Device.(*accessory.ColoredLightbulb).Lightbulb.Brightness.GetValue()
		log.Info.Printf("current brightness: %d, target: %d", bv, target)
		if bv == target {
			log.Info.Printf("adjusting target too 99")
			target = 99
		}

		// update hardware
		_, err := tradfriClient.PutDeviceDimming(a.Name, target*255/100)
		if err != nil {
			log.Info.Println(err.Error())
		}

		// update GUI
		switch a.Device.(type) {
		case *accessory.ColoredLightbulb:
			a.Device.(*accessory.ColoredLightbulb).Lightbulb.Brightness.SetValue(target)
		case *devices.TempLightbulb:
			a.Device.(*devices.TempLightbulb).Lightbulb.Brightness.SetValue(target)
		}
	case "Toggle":
		log.Info.Println("toggle verb called")
	default:
		log.Info.Printf("unknown tradfri verb: %s", action.Verb)
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

		switch tdd.Device.(type) {
		case *accessory.ColoredLightbulb:
			clb := tdd.Device.(*accessory.ColoredLightbulb)
			if clb.Lightbulb.On.GetValue() != (d.LightControl[0].Power > 0) {
				clb.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)
			}
			dv := int(mapRange(float64(d.LightControl[0].Dimmer), 0, 254, 0, 100))
			clb.Lightbulb.Brightness.SetValue(dv)
			dhue := mapRange(float64(d.LightControl[0].Hue), 0, 65279, 0, 360)
			clb.Lightbulb.Hue.SetValue(dhue)
			dsat := mapRange(float64(d.LightControl[0].Saturation), 0, 65279, 0, 100)
			clb.Lightbulb.Saturation.SetValue(dsat)
		case *accessory.Lightbulb:
			lb := tdd.Device.(*accessory.Lightbulb)
			if lb.Lightbulb.On.GetValue() != (d.LightControl[0].Power > 0) {
				lb.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)
			}
		case *devices.TempLightbulb:
			tlb := tdd.Device.(*devices.TempLightbulb)
			if tlb.Lightbulb.On.GetValue() != (d.LightControl[0].Power > 0) {
				tlb.Lightbulb.On.SetValue(d.LightControl[0].Power > 0)
			}
			dv := int(mapRange(float64(d.LightControl[0].Dimmer), 0, 254, 0, 100))
			if tlb.Lightbulb.Brightness.GetValue() != dv {
				tlb.Lightbulb.Brightness.SetValue(dv)
				// if you change the color in Ikea's app, ... well ... I can't convert from HSL to Kelvin yet
			}
		}
	}
}

func kelvinToRGB(k int) (r, g, b float64) {
	if k < 1000 {
		k = 1000
	} else if k > 40000 {
		k = 40000
	}
	t := float64(k / 100)
	if t <= 66 {
		r = 1
		g = bound((99.4708025861*math.Log(t) - 161.1195681661) / 255)
	} else {
		r = bound((329.698727446 * math.Pow(t-60, -0.1332047592)) / 255)
		g = bound((288.1221695283 * math.Pow(t-60, -0.0755148492)) / 255)
	}
	if t >= 66 {
		b = 1
	} else if t <= 19 {
		b = 0
	} else {
		b = bound((138.5177312231*math.Log(t-10) - 305.0447927307) / 255)
	}
	return
}

func bound(f float64) float64 {
	if f <= 0 {
		return 0
	} else if f >= 1 {
		return 1
	} else {
		return f
	}
}
