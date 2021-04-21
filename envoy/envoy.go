package envoy

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/cloudkucooland/go-envoy"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"
	"strconv"
	"sync"
	"time"
)

// Platform is the platform handle for the Kasa stuff
type Platform struct {
	Running bool
}

var envoys map[string]*tfaccessory.TFAccessory
var doOnce sync.Once

// Startup is called by the platform management to start the platform up
func (p Platform) Startup(c *config.Config) platform.Control {
	p.Running = true
	envoy.SetLogger(log.Info)
	return p
}

// Shutdown is called by the platform management to shut things down
func (p Platform) Shutdown() platform.Control {
	p.Running = false
	return p
}

// AddAccessory adds an envoy device, then adds it to HC
func (p Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	doOnce.Do(func() {
		envoys = make(map[string]*tfaccessory.TFAccessory)
	})

	e := envoy.New(a.IP)
	settings, err := e.Info()
	if err != nil {
		log.Info.Println(err)
		return
	}

	a.Type = accessory.TypeSensor
	a.Info.Name = a.Name
	a.Info.SerialNumber = settings.Device.Sn
	a.Info.FirmwareRevision = settings.Device.Software
	a.Info.Model = "IQ Envoy"
	a.Info.Manufacturer = "Enphase"
	id, err := strconv.Atoi(settings.Device.Sn)
	if err != nil {
		a.Info.ID = 9999
	}
	a.Info.ID = uint64(id)

	a.Device = devices.NewEnvoy(a.Info)
	a.Accessory = a.Device.(*devices.Envoy).Accessory

	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)

	envoys[a.IP] = a
	a.Device.(*devices.Envoy).Envoy = e
	log.Info.Printf("Enphase IQ Envoy ID: %s\n", a.Info.SerialNumber)

	// set initial state
	nowprod, nowcon, _, err := e.Now()
	if err != nil {
		log.Info.Println(err.Error())
	}
	if nowprod < nowcon {
		a.Device.(*devices.Envoy).Active.SetValue(characteristic.ActiveInactive)
		a.Device.(*devices.Envoy).LightSensor.CurrentAmbientLightLevel.SetValue(0.0001)
		a.Device.(*devices.Envoy).DailyProduction.ChargingState.SetValue(characteristic.ChargingStateNotCharging)
		a.Device.(*devices.Envoy).DailyConsumption.ChargingState.SetValue(characteristic.ChargingStateCharging)
	} else {
		a.Device.(*devices.Envoy).LightSensor.CurrentAmbientLightLevel.SetValue(nowprod)
		a.Device.(*devices.Envoy).DailyProduction.ChargingState.SetValue(characteristic.ChargingStateCharging)
		a.Device.(*devices.Envoy).DailyConsumption.ChargingState.SetValue(characteristic.ChargingStateNotCharging)
	}

	production, consumption, _, err := e.Today()
	if err != nil {
		log.Info.Println(err.Error())
	}
	production = production / 1000.0
	a.Device.(*devices.Envoy).DailyProduction.BatteryLevel.SetValue(int(production))
	consumption = consumption / 1000.0
	a.Device.(*devices.Envoy).DailyConsumption.BatteryLevel.SetValue(int(consumption))
}

// GetAccessory looks up a device by IP address
func (p Platform) GetAccessory(ip string) (*tfaccessory.TFAccessory, bool) {
	val, ok := envoys[ip]
	return val, ok
}

// Background pulls the envoys every 300 seconds
func (p Platform) Background() {
	go func() {
		for range time.Tick(time.Second * 300) {
			p.backgroundPuller()
		}
	}()
}

func (p Platform) backgroundPuller() {
	for _, a := range envoys {
		nowprod, nowcon, _, err := a.Device.(*devices.Envoy).Envoy.Now()
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		if nowprod < nowcon { // no consumption exceeds production, mark this for for Home.app automation
			a.Device.(*devices.Envoy).LightSensor.CurrentAmbientLightLevel.SetValue(0.0001)
			if a.Device.(*devices.Envoy).Active.GetValue() == characteristic.ActiveActive {
				a.Device.(*devices.Envoy).Active.SetValue(characteristic.ActiveInactive)
				a.Device.(*devices.Envoy).DailyProduction.ChargingState.SetValue(characteristic.ChargingStateNotCharging)
				a.Device.(*devices.Envoy).DailyConsumption.ChargingState.SetValue(characteristic.ChargingStateCharging)
			}
		} else {
			a.Device.(*devices.Envoy).LightSensor.CurrentAmbientLightLevel.SetValue(nowprod)
			if a.Device.(*devices.Envoy).Active.GetValue() == characteristic.ActiveInactive {
				a.Device.(*devices.Envoy).Active.SetValue(characteristic.ActiveActive)
				a.Device.(*devices.Envoy).DailyProduction.ChargingState.SetValue(characteristic.ChargingStateCharging)
				a.Device.(*devices.Envoy).DailyConsumption.ChargingState.SetValue(characteristic.ChargingStateNotCharging)
			}
		}

		production, consumption, _, err := a.Device.(*devices.Envoy).Envoy.Today()
		if err != nil {
			log.Info.Println(err.Error())
		}
		// log.Info.Printf("Daily total: %2.2f", daily)
		production = production / 1000.0
		a.Device.(*devices.Envoy).DailyProduction.BatteryLevel.SetValue(int(production))
		consumption = consumption / 1000.0
		a.Device.(*devices.Envoy).DailyConsumption.BatteryLevel.SetValue(int(consumption))
	}
}
