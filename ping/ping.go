package ping

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/service"
	"github.com/brutella/hc/util"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/platform"
	"net/http"
	"sync"
	"time"
	"fmt"
)

// Platform is the platform handle for the Kasa stuff
type Platform struct {
	Running bool
}

// defined by ping devices
type Device struct {
	Port int
}

var devices map[string]*tfaccessory.TFAccessory
var doOnce sync.Once

// Startup is called by the platform management to start the platform up
func (p Platform) Startup(c *config.Config) platform.Control {
	p.Running = true
	return p
}

// Shutdown is called by the platform management to shut things down
func (p Platform) Shutdown() platform.Control {
	p.Running = false
	return p
}

// AddAccessory adds a ping device, then adds it to HC
func (p Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	storage, err := util.NewFileStorage("serials")
	if err != nil {
		log.Info.Println("unable to get storage")
	}

	doOnce.Do(func() {
		devices = make(map[string]*tfaccessory.TFAccessory)
	})

	if a.Info.Name == "" {
		a.Info.Name = a.Name
	}
	a.Type = accessory.TypeSecuritySystem
	serial := util.GetSerialNumberForAccessoryName(a.Info.Name, storage)
	a.Info.SerialNumber = serial
	a.Info.Model = "TooFarPing"
	a.Info.FirmwareRevision = "0.0.2"
	a.Info.Manufacturer = "deviousness"

	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)

	devices[a.IP] = a

	up := ping(a)
	cs := service.NewContactSensor()
	cs.ContactSensorState.SetValue(up)
	a.Accessory.AddService(cs.Service)

	// why is (was?) this necessary?
	bs := service.NewBridgingState()
	bs.Reachable.SetValue(true)
	bs.Category.SetValue(1)
	bs.LinkQuality.SetValue(1)
	bs.AccessoryIdentifier.SetValue(a.Name)

	a.Accessory.AddService(bs.Service)
}

// GetAccessory looks up a device by IP address
func (p Platform) GetAccessory(ip string) (*tfaccessory.TFAccessory, bool) {
	val, ok := devices[ip]
	return val, ok
}

// Background runs a background Go task periodically pinging everything
func (p Platform) Background() {
	go func() {
		for range time.Tick(time.Second * 60) {
			p.backgroundPuller()
		}
	}()
}

func (p Platform) backgroundPuller() {
	for _, a := range devices {
		up := ping(a)
		cs := getCS(a)
		if cs != nil {
			cs.Value = up
		}
	}
}

func ping(a *tfaccessory.TFAccessory) int {
	client := &http.Client{}

	testurl := fmt.Sprintf("http://%s/", a.IP)
	req, err := http.NewRequest("GET", testurl, nil)
	if err != nil {
		log.Info.Println(err.Error())
		return 1
	}
	_, err = client.Do(req)
	if err != nil {
		log.Info.Println(err.Error())
		return 1
	}
	// defer resp.Body.Close()
	/* _, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Info.Println(err.Error())
		return 1
	} */
	return 0
}

func getCS(a *tfaccessory.TFAccessory) *characteristic.Characteristic {
	for _, v := range a.GetServices() {
		if v.Type == service.TypeContactSensor {
			// log.Info.Printf("%+v", v)
			for _, cv := range v.Characteristics {
				if cv.Type == characteristic.TypeContactSensorState {
					// log.Info.Printf("%+v", cv)
					return cv
				}
			}
		}
	}
	return nil
}
