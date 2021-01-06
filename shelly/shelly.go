package shelly

import (
	tfaccessory "indievisible.org/toofar/accessory"
	"indievisible.org/toofar/action"
	"indievisible.org/toofar/config"
	"indievisible.org/toofar/platform"
	"indievisible.org/toofar/runner"

	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

type settings struct {
	Device   shellyDev     `json:"device"`
	Name     string        `json:"name"`
	Firmware string        `json:"fw",omitempty`
	Relays   []shellyRelay `json:"relays"`
}

// these are minimial versions of just what we need here
type shellyDev struct {
	Type string `json:"type"`
	Mac  string `json:"mac"`
}

// /relay/0
type shellyRelay struct {
	IsOn bool `json:"ison"`
}

// Handler is registered with the HTTP platform
// it listens for shelly devices and respond appropriately
func Handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cmd := vars["cmd"]

	s, ok := platform.GetPlatform("Shelly")
	if !ok {
		log.Info.Print("unable to get shelly platform, giving up")
		http.Error(w, `{ "status": "bad" }`, http.StatusInternalServerError)
		return
	}

	// use LastIndex since ipv6...
	remoteAddr := r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]

	a, ok := s.GetAccessory(remoteAddr)
	if !ok {
		log.Info.Printf("shelly state from unknown device (%s), ignoring", remoteAddr)
		http.Error(w, `{ "status": "bad" }`, http.StatusNotAcceptable)
		return
	}

	log.Info.Printf("from shelly [%s] to me: [%s]", r.RemoteAddr, cmd)
	switch cmd {
	case "on": // turned on at switch, update GUI
		updateHCGUI(a, true)
		actions := a.MatchActions("On")
		runner.RunActions(actions)
	case "off": // turned off at switch, update GUI
		updateHCGUI(a, false)
		actions := a.MatchActions("Off")
		runner.RunActions(actions)
	case "outon": // turned on in software, update the GUI
		if !a.Switch.Switch.On.GetValue() {
			updateHCGUI(a, true)
		}
		actions := a.MatchActions("OutOn")
		runner.RunActions(actions)
	case "outoff": // turned on in software, update the GUI
		if a.Switch.Switch.On.GetValue() {
			updateHCGUI(a, false)
		}
		actions := a.MatchActions("OutOff")
		runner.RunActions(actions)
	default:
		log.Info.Printf("unknown shelly command: %s", cmd, r.RemoteAddr)
		actions := a.MatchActions("default")
		runner.RunActions(actions)
	}

	fmt.Fprint(w, `{ "status": "OK" }`)
}

// Platform is the handle to the shelly devices
type Platform struct {
	Running bool
}

var shellies map[string]*tfaccessory.TFAccessory
var doOnceShelly sync.Once

// Startup is called by the platform management to get things going
func (s Platform) Startup(c config.Config) platform.Control {
	s.Running = true
	return s
}

// Shutdown is called by the platform management to shut things down
func (s Platform) Shutdown() platform.Control {
	s.Running = false
	return s
}

// AddAccessory adds a Shelly device and registers it with HC
func (s Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	doOnceShelly.Do(func() {
		shellies = make(map[string]*tfaccessory.TFAccessory)
	})

	a.Type = accessory.TypeSwitch

	// pull shelly to get a.Info -- override the config file with reality
	settings, err := getSettings(a)
	if err != nil {
		log.Info.Print("unable to identify Shelly Device: %s", err.Error())
		return
	}
	a.Info.Name = settings.Name
	a.Info.SerialNumber = settings.Device.Mac
	a.Info.Manufacturer = "Shelly"
	a.Info.Model = settings.Device.Type
	a.Info.FirmwareRevision = settings.Firmware

	// convert the Mac address into a uint64 for the ID
	mac, err := hex.DecodeString(settings.Device.Mac)
	if err != nil {
		log.Info.Printf("weird shelly MAC: %s", err.Error())
	}
	for k, v := range mac {
		a.Info.ID += uint64(v) << (12 - k) * 8
	}

	// shellies are indexed by IP address
	shellies[a.IP] = a

	// add to HC for GUI
	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)
	updateHCGUI(a, settings.Relays[0].IsOn)

	// install callback: if we get an update from HC, deal with it
	a.Switch.Switch.On.OnValueRemoteUpdate(func(newstate bool) {
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
	})

	a.Runner = shellyRunner
}

func shellyRunner(a *tfaccessory.TFAccessory, action *action.Action) {
	log.Info.Printf("in shelly action runner: %+v", a)
}

func updateHCGUI(a *tfaccessory.TFAccessory, newstate bool) {
	log.Info.Printf("setting Shelly [%s] HC GUI to: %t", a.Name, newstate)
	if a.Switch != nil {
		a.Switch.Switch.On.SetValue(newstate)
	}
}

func setState(a *tfaccessory.TFAccessory, newstate bool) (*shellyRelay, error) {
	client := &http.Client{}

	grr := "off"
	if newstate {
		grr = "on"
	}
	log.Info.Printf("setting Shelly hardware [%s] to: %s", a.Name, grr)
	relayurl := fmt.Sprintf("http://%s/relay/0?turn=%s", a.IP, grr)
	req, err := http.NewRequest("GET", relayurl, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(a.Username, a.Password)
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

	var r shellyRelay
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func getState(a *tfaccessory.TFAccessory) (*shellyRelay, error) {
	client := &http.Client{}

	url := fmt.Sprintf("http://%s/relay/0", a.IP)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(a.Username, a.Password)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r shellyRelay
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// GetAccessory looks up a Shelly device by IP address
func (s Platform) GetAccessory(ip string) (*tfaccessory.TFAccessory, bool) {
	val, ok := shellies[ip]
	return val, ok
}

func getSettings(a *tfaccessory.TFAccessory) (*settings, error) {
	client := &http.Client{}

	url := fmt.Sprintf("http://%s/settings", a.IP)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(a.Username, a.Password)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// log.Info.Printf("%+v", string(body))

	var sd settings
	if err := json.Unmarshal(body, &sd); err != nil {
		return nil, err
	}
	return &sd, nil
}

// Background starts up the go process to periodically verify the shelly's state
func (s Platform) Background() {
	go func() {
		for range time.Tick(time.Second * 30) {
			s.backgroundPuller()
		}
	}()
}

func (s Platform) backgroundPuller() {
	for _, a := range shellies {
		r, err := getState(a)
		if err != nil {
			log.Info.Println(err.Error())
			continue
		}
		if a.Switch.Switch.On.GetValue() != r.IsOn {
			updateHCGUI(a, r.IsOn)
		}
	}
}
