package konnected

import (
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/action"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"
	"github.com/cloudkucooland/toofar/runner"

	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type system struct {
	Mac       string     `json:"mac"`
	IP        string     `json:"ip",omitempty`
	Gateway   string     `json:"gw",omitempty`
	Netmask   string     `json:"nm",omitempty`
	Hardware  string     `json:"hwVersion",omitempty`
	RSSI      int8       `json:"rssi",omitempty`
	Software  string     `json:"swVersion",omitempty`
	Port      uint16     `json:"port",omitempty`
	Uptime    uint64     `json:"uptime",omitempty`
	Heap      uint64     `json:"heap",omitempty`
	Settings  settings   `json:"settings"`
	Sensors   []sensor   `json:"sensors"`
	DBSensors []sensor   `json:"ds18b20_sensors"`
	Actuators []actuator `json:"actuators"`
	DHTs      []dht      `json:"dht_sensors"`
}

type settings struct {
	EndpointType string `json:"endpoint_type",omitempty`
	Endpoint     string `json:"endpoint",omitempty`
	Token        string `json:"token",omitempty`
}

type sensor struct {
	Pin   uint8 `json:"pin"`
	State uint8 `json:"state"`
	Retry uint8 `json:"retry",omitempty`
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
// if the board doesn't get a 200 in response, it retries, and failing several retries, it reboots
// we will just say OK no matter what for now
func Handler(w http.ResponseWriter, r *http.Request) {
	s, ok := platform.GetPlatform("Konnected")
	if !ok {
		log.Info.Print("unable to get konnected platform, giving up")
		// http.Error(w, `{ "status": "bad" }`, http.StatusInternalServerError)
		// acknowledge so it doesn't retransmit
		fmt.Fprint(w, `{ "status": "OK" }`)
		return
	}

	vars := mux.Vars(r)
	device := vars["device"]
	a, ok := s.GetAccessory(device)
	if !ok {
		log.Info.Printf("konnected state from unknown device (%s / %s), ignoring", r.RemoteAddr, device)
		// http.Error(w, `{ "status": "bad" }`, http.StatusNotAcceptable)
		fmt.Fprint(w, `{ "status": "OK" }`)
		return
	}

	// verify token, if set in local config
	if a.Password != "" {
		sentToken := r.Header.Get("Authorization")
		if sentToken == "" {
			log.Info.Printf("Authorization token not sent")
			// http.Error(w, `{ "status": "bad" }`, http.StatusForbidden)
			fmt.Fprint(w, `{ "status": "OK" }`)
			return
		}
		if sentToken[7:] != a.Password {
			log.Info.Printf("Authorization token invalid")
			// http.Error(w, `{ "status": "bad" }`, http.StatusForbidden)
			fmt.Fprint(w, `{ "status": "OK" }`)
			return
		}
	}

	jBlob, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Info.Printf("konnected: unable to read update")
		// http.Error(w, `{ "status": "bad" }`, http.StatusInternalServerError)
		fmt.Fprint(w, `{ "status": "OK" }`)
		return
	}
	// if konnected provisioned with a trailing / on the url..
	if string(jBlob) == "" {
		log.Info.Printf("konnected: sent empty message")
		// acknowledge the notice so it doesn't retransmit
		fmt.Fprint(w, `{ "status": "OK" }`)
		// trigger a manual pull
		err := getStatusAndUpdate(a)
		if err != nil {
			log.Info.Println(err.Error())
		}
		return
	}

	var p sensor
	// log.Info.Printf("sent from %+v: %s", a.Name, string(jBlob))
	err = json.Unmarshal(jBlob, &p)
	if err != nil {
		log.Info.Printf("konnected: unable to understand update")
		// http.Error(w, `{ "status": "bad" }`, http.StatusNotAcceptable)
		fmt.Fprint(w, `{ "status": "OK" }`)
		return
	}

	// tell homekit about the change
	if svc, ok := a.Device.(*devices.Konnected).Pins[p.Pin]; ok {
		switch svc.(type) {
		case devices.KonnectedSystem:
			// system pin changed
			newVal := characteristic.SecuritySystemCurrentStateAwayArm
			if p.State == 0 {
				newVal = characteristic.SecuritySystemCurrentStateDisarmed
			}
			a.Device.(*devices.Konnected).SecuritySystem.SecuritySystemCurrentState.SetValue(newVal)
		case *devices.KonnectedMotionSensor:
			svc.(*devices.KonnectedMotionSensor).MotionDetected.SetValue(p.State == 1)
		case *devices.KonnectedContactSensor:
			svc.(*devices.KonnectedContactSensor).ContactSensorState.SetValue(int(p.State))
		default:
			log.Info.Println("bad type in handler: %+v", svc)
		}
	}
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
	if timeout == 0 {
		timeout = 10
	}

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

	details, err := getDetails(a)
	if err != nil {
		log.Info.Printf("unable to identify Konnected device: %s", err.Error())
		return
	}
	a.Info.Name = a.Name
	a.Info.SerialNumber = a.Username
	a.Info.Manufacturer = "Konnected.io"
	a.Info.Model = details.Hardware
	a.Info.FirmwareRevision = details.Software

	// convert the Mac address into a uint64 for the ID
	mac, err := hex.DecodeString(a.Username) // details.Mac
	if err != nil {
		log.Info.Printf("weird konnected MAC: %s", err.Error())
	}
	for k, v := range mac {
		a.Info.ID += uint64(v) << (12 - k) * 8
	}

	// konnecteds are indexed by device IDs
	konnecteds[a.Info.SerialNumber] = a

	a.Device = devices.NewKonnected(a.Info)
	a.Accessory = a.Device.(*devices.Konnected).Accessory

	for _, v := range a.KonnectedZones {
		switch v.Type {
		case "system":
			p := devices.KonnectedSystem{}
			a.Device.(*devices.Konnected).Pins[v.Pin] = p
			// no need for an HK display
			log.Info.Printf("Konnected Pin: %d: %s (system)", v.Pin, v.Name)
		case "motion":
			p := devices.NewKonnectedMotionSensor(v.Name)
			a.Device.(*devices.Konnected).Pins[v.Pin] = p
			a.Accessory.AddService(p.Service)
			log.Info.Printf("Konnected Pin: %d: %s (motion)", v.Pin, v.Name)
		case "door":
			p := devices.NewKonnectedContactSensor(v.Name)
			a.Device.(*devices.Konnected).Pins[v.Pin] = p
			a.Accessory.AddService(p.Service)
			log.Info.Printf("Konnected Pin: %d: %s (contact)", v.Pin, v.Name)
		default:
			log.Info.Println("unknown KonnectedZone type")
		}
	}

	for _, v := range details.Sensors {
		if p, ok := a.Device.(*devices.Konnected).Pins[v.Pin]; ok {
			switch p.(type) {
			case *devices.KonnectedSystem:
				// system pin changed
				newVal := characteristic.SecuritySystemCurrentStateAwayArm
				if v.State == 0 {
					newVal = characteristic.SecuritySystemCurrentStateDisarmed
				}
				a.Device.(*devices.Konnected).SecuritySystem.SecuritySystemCurrentState.SetValue(newVal)
			case *devices.KonnectedContactSensor:
				p.(*devices.KonnectedContactSensor).ContactSensorState.SetValue(int(v.State))
			case *devices.KonnectedMotionSensor:
				p.(*devices.KonnectedMotionSensor).MotionDetected.SetValue(v.State == 1)
			}
		}
	}

	a.Device.(*devices.Konnected).SecuritySystem.SecuritySystemTargetState.OnValueRemoteUpdate(func(newval int) {
		// do the work to adjust the state
		log.Info.Printf("HC requested system state change to %d", newval)
		a.Device.(*devices.Konnected).SecuritySystem.SecuritySystemCurrentState.SetValue(newval)
		switch newval {
		case characteristic.SecuritySystemCurrentStateStayArm:
			actions := a.MatchActions("Home")
			runner.RunActions(actions)
		case characteristic.SecuritySystemCurrentStateAwayArm:
			actions := a.MatchActions("Away")
			runner.RunActions(actions)
		case characteristic.SecuritySystemCurrentStateNightArm:
			actions := a.MatchActions("Night")
			runner.RunActions(actions)
		case characteristic.SecuritySystemCurrentStateDisarmed:
			actions := a.MatchActions("Disarmed")
			runner.RunActions(actions)
		case characteristic.SecuritySystemCurrentStateAlarmTriggered:
			actions := a.MatchActions("Triggered")
			runner.RunActions(actions)
		default:
			log.Info.Printf("unknown security system state: %d", newval)
		}
	})

	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)

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

func (s Platform) GetAccessory(mac string) (*tfaccessory.TFAccessory, bool) {
	val, ok := konnecteds[mac]
	return val, ok
}

func getDetails(a *tfaccessory.TFAccessory) (*system, error) {
	url := fmt.Sprintf("http://%s/status", a.IP)
	body, err := doRequest(a, "GET", url)
	if err != nil {
		return nil, err
	}

	var s system
	if err := json.Unmarshal(*body, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func getStatus(a *tfaccessory.TFAccessory) (*[]sensor, error) {
	url := fmt.Sprintf("http://%s/device", a.IP)
	body, err := doRequest(a, "GET", url)
	if err != nil {
		return nil, err
	}

	var s []sensor
	if err := json.Unmarshal(*body, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func getStatusAndUpdate(a *tfaccessory.TFAccessory) error {
	status, err := getStatus(a)
	if err != nil {
		return err
	}

	for _, v := range *status {
		if p, ok := a.Device.(*devices.Konnected).Pins[v.Pin]; ok {
			switch p.(type) {
			// case "system":
			case *devices.KonnectedMotionSensor:
				p.(*devices.KonnectedMotionSensor).MotionDetected.SetValue(v.State == 1)
			case *devices.KonnectedContactSensor:
				if p.(*devices.KonnectedContactSensor).ContactSensorState.GetValue() != int(v.State) {
					p.(*devices.KonnectedContactSensor).ContactSensorState.SetValue(int(v.State))
				}
			}
		}
	}
	return nil
}

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
		err := getStatusAndUpdate(a)
		if err != nil {
			log.Info.Println(err.Error())
			continue
		}
	}
}
