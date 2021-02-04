package konnected

import (
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"
	// "github.com/cloudkucooland/toofar/runner"

	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	// "strings"
	"sync"
	"time"
)

type settings struct {
	Mac          string
	Name         string
	IsOn         int // this is not correct, I assume it will be one of the sensors
	Device       *devices.Konnected
	Firmware     string
	EndpointType string     `json:"endpoint_type",omitempty`
	Endpoint     string     `json:"endpoint",omitempty`
	Token        string     `json:"token",omitempty`
	Sensors      []sensor   `json:"sensors",omitempty`
	Actuators    []actuator `json:"actuators",omitempty`
	DHTs         []dht      `json:"dht_sensors",omitempty`
}

type sensor struct {
	Pin    uint8 `json:"pin"`
	Invert bool
}

type actuator struct {
	Pin     uint8 `json:"pin"`
	Trigger uint8 `json:"trigger"`
}

type dht struct {
	Pin  uint8 `json:"pin"`
	Poll uint  `json:"poll_interval"`
}

type command struct {
	Pin       uint8  `json:"pin"`
	State     uint8  `json:"state"`
	Momentary uint16 `json:"state",omitempty`
	Times     uint8  `json:"times",omitempty`
	Pause     uint8  `json:"pause",omitempty`
}

// Handler is registered with the HTTP platform
// it listens for Konnected devices and respond appropriately
func Handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	device := vars["device"]

	s, ok := platform.GetPlatform("Konnected")
	if !ok {
		log.Info.Print("unable to get konnected platform, giving up")
		http.Error(w, `{ "status": "bad" }`, http.StatusInternalServerError)
		return
	}

	// index these by mac address w/o :
	a, ok := s.GetAccessory(device)
	if !ok {
		// log.Info.Printf("shelly state from unknown device (%s), ignoring", remoteAddr)
		http.Error(w, `{ "status": "bad" }`, http.StatusNotAcceptable)
		return
	}
	k := a.Device.(*devices.Konnected)
	log.Info.Printf("%+v", k)

	// do stuff here
	fmt.Fprint(w, `{ "status": "OK" }`)
}

// Platform is the handle to the Konnected devices
type Platform struct {
	Running bool
}

var konnecteds map[string]*tfaccessory.TFAccessory
var doOnce sync.Once
var client *http.Client

// Startup is called by the platform management to get things going
func (s Platform) Startup(c *config.Config) platform.Control {
	s.Running = true

	timeout := config.Get().KonnectedTimeout
	// unset, default to something reasonable
	if timeout == 0 {
		timeout = 10
	}

	// these values are aggressive, probably not good for sites with lots of shellies
	tr := &http.Transport{
		MaxIdleConns:    5,
		IdleConnTimeout: 30 * time.Second,
	}
	client = &http.Client{Transport: tr, Timeout: time.Second * time.Duration(timeout)}
	return s
}

// Shutdown is called by the platform management to shut things down
func (s Platform) Shutdown() platform.Control {
	s.Running = false
	return s
}

// AddAccessory adds a Konnected device and registers it with HC
func (s Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	doOnce.Do(func() {
		konnecteds = make(map[string]*tfaccessory.TFAccessory)
	})

	a.Type = accessory.TypeSecuritySystem

	settings, err := getSettings(a)
	if err != nil {
		log.Info.Printf("unable to identify Konnected device: %s", err.Error())
		return
	}
	a.Info.Name = settings.Name
	a.Info.SerialNumber = a.Username
	a.Info.Manufacturer = "Konnected.io"
	a.Info.Model = "something"
	a.Info.FirmwareRevision = settings.Firmware

	// convert the Mac address into a uint64 for the ID
	mac, err := hex.DecodeString(a.Username)
	if err != nil {
		log.Info.Printf("weird shelly MAC: %s", err.Error())
	}
	for k, v := range mac {
		a.Info.ID += uint64(v) << (12 - k) * 8
	}

	// konnecteds are indexed by device IDs
	konnecteds[a.Info.SerialNumber] = a

	// add to HC for GUI
	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)

	// update UI to reflect the current state

	// sw := a.Device.(*accessory.Switch)
	// install callback: if we get an update from HC, deal with it
	/* sw.Switch.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting [%s] to [%t] from HC handler", a.Name, newstate)
		state, err := setState(a, newstate)
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
		if state.IsOn != newstate {
			log.Info.Printf("unable to update shelly state to %t", newstate)
			updateHCGUI(a, state.IsOn)
		}
	}) */

	a.Runner = kRunner
}

func kRunner(a *tfaccessory.TFAccessory, action *action.Action) {
	log.Info.Printf("in konnected action runner: %+v", a)
}

func doRequest(a *tfaccessory.TFAccessory, method, url string) (*[]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// req.SetBasicAuth(a.Username, a.Password)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Info.Println(err.Error())
		return nil, err
	}
	return &body, nil
}

// GetAccessory looks up a Shelly device by IP address
func (s Platform) GetAccessory(ip string) (*tfaccessory.TFAccessory, bool) {
	val, ok := konnecteds[ip]
	return val, ok
}

/* mappings {
  path("/device/:mac/:id/:deviceState") { action: [ PUT: "childDeviceStateUpdate"] }
  path("/device/:mac") { action: [ PUT: "childDeviceStateUpdate", GET: "getDeviceState" ] }
  path("/ping") { action: [ GET: "devicePing"] }
}

https://github.com/konnected-io/homebridge-konnected/blob/master/src/platform.ts
*/
func getSettings(a *tfaccessory.TFAccessory) (*settings, error) {
	url := fmt.Sprintf("http://%s/device/%s", a.IP, a.Username)
	body, err := doRequest(a, "GET", url)
	if err != nil {
		return nil, err
	}
	var sd settings
	if err := json.Unmarshal(*body, &sd); err != nil {
		return nil, err
	}
	return &sd, nil
}

// Background starts up the go process to periodically verify the shelly's state
func (k Platform) Background() {
	kpr := config.Get().KonnectedPullRate
	if kpr == 0 {
		log.Info.Println("pull rate set to 0, disabling konnected puller")
	}
	go func() {
		for range time.Tick(time.Second * time.Duration(kpr)) {
			k.backgroundPuller()
		}
	}()
}

func (k Platform) backgroundPuller() {
	for _, a := range konnecteds {
		settings, err := getSettings(a)
		if err != nil {
			log.Info.Println(err.Error())
			continue
		}
		log.Info.Printf("%+v", settings)
		if a.Device.(*devices.Konnected).SecuritySystem.SecuritySystemCurrentState.GetValue() != settings.IsOn {
			a.Device.(*devices.Konnected).SecuritySystem.SecuritySystemCurrentState.SetValue(settings.IsOn)
		}
	}
}
