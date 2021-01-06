package ping

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/service"
	tfaccessory "indievisible.org/toofar/accessory"
	"indievisible.org/toofar/config"
	"indievisible.org/toofar/platform"
	"net/http"
	"sync"
	"time"
	// "io/ioutil"
	"fmt"
	goping "github.com/go-ping/ping"
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
func (p Platform) Startup(c config.Config) platform.Control {
	p.Running = true
	return p
}

// Shutdown is called by the platform management to shut things down
func (p Platform) Shutdown() platform.Control {
	p.Running = false
	return p
}

// AddAccessory adds a Kasa device, pulls it for info, then adds it to HC
func (p Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	doOnce.Do(func() {
		devices = make(map[string]*tfaccessory.TFAccessory)
	})

	if a.Info.Name == "" {
		a.Info.Name = a.Name
	}
	a.Type = accessory.TypeSecuritySystem

	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)

	devices[a.IP] = a

	up := ping(a)
	cs := service.NewContactSensor()
	cs.ContactSensorState.SetValue(up)
	a.Accessory.AddService(cs.Service)
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

func oldping(a *tfaccessory.TFAccessory) int {
	pinger, err := goping.NewPinger(a.IP)
	// pinger.SetPrivileged(true)
	// requires: setcap cap_net_raw=+ep /path/to/your/compiled/binary
	if err != nil {
		log.Info.Println(err.Error())
		return 1
	}
	pinger.Count = 3
	err = pinger.Run()
	if err != nil {
		log.Info.Println(err.Error())
		return 1
	}
	if pinger.PacketsSent != pinger.PacketsRecv {
		log.Info.Printf("Sent %d, Rec: %d", pinger.PacketsSent, pinger.PacketsRecv)
		return 1
	}
	return 0
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
