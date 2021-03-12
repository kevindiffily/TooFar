package kasa

import (
	"bytes"
	"encoding/hex"
	// "encoding/json"
	"fmt"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	// "github.com/brutella/hc/util"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"
	"github.com/cloudkucooland/toofar/runner"
	"net"
	"sync"
	"time"
)

// https://www.softscheck.com/en/reverse-engineering-tp-link-hs110/#TP-Link%20Smart%20Home%20Protocol
// https://medium.com/@hu3vjeen/reverse-engineering-tp-link-kc100-bac4641bf1cd
// https://machinekoder.com/controlling-tp-link-hs100110-smart-plugs-with-machinekit/
// https://lib.dr.iastate.edu/cgi/viewcontent.cgi?article=1424&context=creativecomponents
// ahttps://github.com/p-doyle/Python-KasaSmartPowerStrip

// see if we can support these...
// https://github.com/brutella/hc/blob/master/characteristic/program_mode.go
// https://github.com/brutella/hc/blob/master/characteristic/set_duration.go
// https://github.com/brutella/hc/blob/master/characteristic/remaining_duration.go

const (
	cmd_sysinfo     = `{"system":{"get_sysinfo":{}}}`
	cmd_countdown   = `{"count_down":{"get_rules":{}}}`
	broadcast_sends = 1
)

// Platform is the platform handle for the Kasa stuff
type Platform struct {
	Running bool
}

type kmu struct {
	mu sync.Mutex
	ks map[string]*tfaccessory.TFAccessory
}

var kasas kmu
var doOnce sync.Once
var kasaUDPconn *net.UDPConn

// Startup is called by the platform management to start the platform up
func (k Platform) Startup(c *config.Config) platform.Control {
	udpl, err := net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: 9999})
	if err != nil {
		fmt.Printf("unable to start UDP listener: %s", err.Error())
		return k
	}
	kasaUDPconn = udpl

	go func() {
		buffer := make([]byte, 1024)
		fmt.Println("starting listener")
		for {
			n, addr, err := kasaUDPconn.ReadFromUDP(buffer)
			if err != nil {
				fmt.Println(err.Error())
				break
			}
			res := decrypt(buffer[0:n])
			doUDPresponse(addr.IP.String(), res)
		}
		// return
	}()

	k.Running = true
	return k
}

// Shutdown is called by the platform management to shut things down
func (k Platform) Shutdown() platform.Control {
	k.Running = false
	return k
}

// AddAccessory adds a Kasa device, pulls it for info, then adds it to HC
func (k Platform) AddAccessory(a *tfaccessory.TFAccessory) {
	doOnce.Do(func() {
		kasas.mu.Lock()
		kasas.ks = make(map[string]*tfaccessory.TFAccessory)
		kasas.mu.Unlock()
	})

	hc, ok := platform.GetPlatform("HomeControl")
	if !ok {
		log.Info.Println("can't add accessory, HomeControl platform does not yet exist")
		return
	}

	_, ok = k.GetAccessory(a.IP)
	if ok {
		log.Info.Printf("already have a device with this IP address: %s", a.IP)
		return
	}

	// override the config file with reality
	settings, err := getSettingsTCP(a)
	if err != nil {
		log.Info.Printf("unable to identify kasa device, skipping: %s", err.Error())
		return
	}
	a.Info.Name = settings.Alias
	a.Info.SerialNumber = settings.DeviceID
	a.Info.Manufacturer = "TP-Link"
	a.Info.Model = settings.Model
	a.Info.FirmwareRevision = settings.SWVersion

	// convert 12 chars of the deviceId into a uint64 for the ID
	mac, err := hex.DecodeString(settings.DeviceID[:12])
	if err != nil {
		log.Info.Printf("weird kasa devid: %s", err.Error())
	}
	for k, v := range mac {
		a.Info.ID += uint64(v) << (12 - k) * 8
	}

	switch settings.Model {
	case "KP303(US)":
		a.Type = accessory.TypeSwitch
		kp := devices.NewKP303(a.Info)
		a.Device = kp
		a.Accessory = kp.Accessory
		a.Runner = runner.GenericSwitchActionRunner
		for i := 0; i < len(kp.Outlets); i++ {
			kp.Outlets[i].On.OnValueRemoteUpdate(func(newval bool) {
				if newval {
					actions := a.MatchActions("On")
					runner.RunActions(actions)
				} else {
					actions := a.MatchActions("Off")
					runner.RunActions(actions)
				}
			})
		}
	case "HS200(US)", "HS210(US)":
		a.Type = accessory.TypeSwitch
		d := devices.NewHS200(a.Info)
		a.Device = d
		a.Accessory = d.Accessory
		a.Runner = runner.GenericSwitchActionRunner
		d.Switch.On.OnValueRemoteUpdate(func(newval bool) {
			if newval {
				actions := a.MatchActions("On")
				runner.RunActions(actions)
			} else {
				actions := a.MatchActions("Off")
				runner.RunActions(actions)
			}
		})
	case "HS103(US)":
		a.Type = accessory.TypeSwitch
		d := devices.NewHS103(a.Info)
		a.Device = d
		a.Accessory = d.Accessory
		a.Runner = runner.GenericSwitchActionRunner
		d.Outlet.On.OnValueRemoteUpdate(func(newval bool) {
			if newval {
				actions := a.MatchActions("On")
				runner.RunActions(actions)
			} else {
				actions := a.MatchActions("Off")
				runner.RunActions(actions)
			}
		})
	case "KP115(US)":
		a.Type = accessory.TypeSwitch
		d := devices.NewKP115(a.Info)
		a.Device = d
		a.Accessory = d.Accessory
		a.Runner = runner.GenericSwitchActionRunner
		d.Outlet.On.OnValueRemoteUpdate(func(newval bool) {
			if newval {
				actions := a.MatchActions("On")
				runner.RunActions(actions)
			} else {
				actions := a.MatchActions("Off")
				runner.RunActions(actions)
			}
		})
	case "HS220(US)":
		a.Type = accessory.TypeLightbulb
		hs := devices.NewHS220(a.Info)
		a.Device = hs
		a.Accessory = hs.Accessory
	}

	log.Info.Printf("adding [%s]: [%s]", a.Info.Name, a.Info.Model)
	// add to HC for GUI
	hc.AddAccessory(a)
	a.Accessory.Info.Name.OnValueRemoteUpdate(func(newname string) {
		log.Info.Print("setting alias to [%s]", newname)
		err := setRelayAlias(a, newname)
		if err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	// kasas are indexed by IP address
	kasas.mu.Lock()
	kasas.ks[a.IP] = a
	kasas.mu.Unlock()

	switch a.Device.(type) {
	case *accessory.Switch:
		// startup value
		sw := a.Device.(*accessory.Switch)
		sw.Switch.On.SetValue(settings.RelayState > 0)

		// install callbacks: if we get an update from HC, deal with it
		sw.Switch.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s] to [%t] from Kasa generic switch handler", a.Name, newstate)
			err := setRelayState(a, newstate)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
		})
	case *devices.HS200: // and 210
		sw := a.Device.(*devices.HS200)
		sw.Switch.On.SetValue(settings.RelayState > 0)
		sw.Switch.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s] to [%t] from HS200 handler", a.Name, newstate)
			err := setRelayState(a, newstate)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
		})
	case *devices.KP115:
		kp := a.Device.(*devices.KP115)
		kp.Outlet.On.SetValue(settings.RelayState > 0)
		kp.Outlet.OutletInUse.SetValue(settings.RelayState > 0)

		kp.Outlet.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s] to [%t] from KP115 handler", a.Name, newstate)
			err := setRelayState(a, newstate)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			kp.Outlet.OutletInUse.SetValue(newstate)
		})
	case *devices.HS103:
		hs := a.Device.(*devices.HS103)
		hs.Outlet.On.SetValue(settings.RelayState > 0)
		hs.Outlet.OutletInUse.SetValue(settings.RelayState > 0)

		hs.Outlet.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s] to [%t] from HS103 handler", a.Name, newstate)
			err := setRelayState(a, newstate)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			hs.Outlet.OutletInUse.SetValue(newstate)
		})
	case *devices.HS220:
		hs := a.Device.(*devices.HS220)
		hs.Lightbulb.On.SetValue(settings.RelayState > 0)
		hs.Lightbulb.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s] to [%t] from HS220 handler", a.Name, newstate)
			err := setRelayState(a, newstate)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
		})
		hs.Lightbulb.Brightness.SetValue(settings.Brightness)
		hs.Lightbulb.Brightness.OnValueRemoteUpdate(func(newval int) {
			log.Info.Printf("setting [%s] brightness [%d] from HS220 handler", a.Name, newval)
			err := setBrightness(a, newval)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
		})
		hs.Lightbulb.SetDuration.OnValueRemoteUpdate(func(newval int) {
			if hs.Lightbulb.ProgramMode.GetValue() != characteristic.ProgramModeNoProgramScheduled {
				log.Info.Println("a countdown is already active, ignoring request")
				return
			}
			log.Info.Println("setting up countdown action")
			current := hs.Lightbulb.On.GetValue()
			err := setProgramState(a, !current, newval)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			hs.Lightbulb.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
		})
	case *devices.KP303:
		kp := a.Device.(*devices.KP303)
		for i := 0; i < len(kp.Outlets); i++ {
			n := characteristic.NewName()
			n.SetValue(settings.Children[i].Alias)
			outlet := kp.Outlets[i]
			outlet.AddCharacteristic(n.Characteristic)

			l := i // local-only copy for this func
			n.OnValueRemoteUpdate(func(newname string) {
				log.Info.Print("setting alias to [%s]", newname)
				err := setChildRelayAlias(a, settings.Children[l].ID, newname)
				if err != nil {
					log.Info.Println(err.Error())
					return
				}
			})

			outlet.On.SetValue(settings.Children[i].RelayState > 0)
			outlet.OutletInUse.SetValue(settings.Children[i].RelayState > 0)
			outlet.On.OnValueRemoteUpdate(func(newstate bool) {
				log.Info.Printf("setting [%s].[%d] to [%t] from KP303 handler", a.Name, l, newstate)
				err := setChildRelayState(a, settings.Children[l].ID, newstate)
				if err != nil {
					log.Info.Println(err.Error())
					return
				}
			})
		}
	}
}

func setRelayState(a *tfaccessory.TFAccessory, newstate bool) error {
	state := 0
	if newstate {
		state = 1
	}
	cmd := fmt.Sprintf(`{"system":{"set_relay_state":{"state":%d}}}`, state)
	err := sendUDP(a.IP, cmd)
	if err != nil {
		log.Info.Println(err.Error())
		return err
	}
	return nil
}

func setChildRelayState(a *tfaccessory.TFAccessory, childID string, newstate bool) error {
	state := 0
	if newstate {
		state = 1
	}
	cmd := fmt.Sprintf(`{"context":{"child_ids":["%s"]},"system":{"set_relay_state":{"state":%d}}}`, childID, state)
	err := sendUDP(a.IP, cmd)
	if err != nil {
		log.Info.Println(err.Error())
		return err
	}
	return nil
}

func setProgramState(a *tfaccessory.TFAccessory, target bool, t int) error {
	err := sendUDP(a.IP, `{"count_down":{"get_rules":{}}}`)
	if err != nil {
		log.Info.Println(err.Error())
		return err
	}

	var state uint8
	if target {
		state = 1
	}
	cmd := fmt.Sprintf(`{"count_down":{"add_rule":{"enable":1,"delay":%d,"act":%d,"name":"TooFar"}}}`, t, state)
	err = sendUDP(a.IP, cmd)
	if err != nil {
		log.Info.Println(err.Error())
		return err
	}
	return nil
}

func deleteCountdown(a *tfaccessory.TFAccessory) error {
	err := sendUDP(a.IP, `{"count_down":{"delete_all_rules":{}}}`)
	if err != nil {
		log.Info.Println(err.Error())
		return err
	}
	return nil
}

func setRelayAlias(a *tfaccessory.TFAccessory, newname string) error {
	cmd := fmt.Sprintf(`{"system":{"set_alias":{"alias":"%s"}}}`, newname)
	err := sendUDP(a.IP, cmd)
	if err != nil {
		log.Info.Println(err.Error())
		return err
	}
	return nil
}

func setChildRelayAlias(a *tfaccessory.TFAccessory, childID string, newname string) error {
	cmd := fmt.Sprintf(`{"context":{"child_ids":["%s"]},"system":{"set_alias":{"alias":"%s"}}}`, childID, newname)
	err := sendUDP(a.IP, cmd)
	if err != nil {
		log.Info.Println(err.Error())
		return err
	}
	return nil
}

func setBrightness(a *tfaccessory.TFAccessory, newval int) error {
	cmd := fmt.Sprintf(`{"smartlife.iot.dimmer":{"set_brightness":{"brightness":%d}}}`, newval)
	err := sendUDP(a.IP, cmd)
	if err != nil {
		log.Info.Println(err.Error())
		return err
	}
	return nil
}

// GetAccessory looks up a Kasa device by IP address
func (k Platform) GetAccessory(ip string) (*tfaccessory.TFAccessory, bool) {
	kasas.mu.Lock()
	val, ok := kasas.ks[ip]
	kasas.mu.Unlock()
	return val, ok
}

func (k Platform) Discover() error {
	return broadcastCmd(cmd_sysinfo)
}

func getSettingBroadcast() error {
	return broadcastCmd(cmd_sysinfo)
}

func getCountdownBroadcast() error {
	return broadcastCmd(cmd_countdown)
}

func broadcastCmd(cmd string) error {
	bcast, err := broadcastAddresses()
	if err != nil {
		return err
	}

	for i := 0; i < broadcast_sends; i++ {
		for _, b := range bcast {
			err := sendUDP(b.String(), cmd)
			if err != nil {
				log.Info.Println(err.Error())
				return err
			}
		}
		time.Sleep(time.Second)
	}
	return nil
}

// Background runs a background Go task verifying HC has the current state of the Kasa devices
func (k Platform) Background() {
	kpr := config.Get().KasaPullRate
	if kpr == 0 {
		log.Info.Println("KasaPullRate is 0, disabling checks")
		return
	}
	go func() {
		for range time.Tick(time.Second * time.Duration(kpr)) {
			getSettingBroadcast()
			getCountdownBroadcast()
		}
	}()
}

func encrypt(plaintext string) []byte {
	n := len(plaintext)
	buf := new(bytes.Buffer)
	ciphertext := []byte(buf.Bytes())

	key := byte(0xAB)
	payload := make([]byte, n)
	for i := 0; i < n; i++ {
		payload[i] = plaintext[i] ^ key
		key = payload[i]
	}

	for i := 0; i < len(payload); i++ {
		ciphertext = append(ciphertext, payload[i])
	}

	return ciphertext
}

func decrypt(ciphertext []byte) string {
	n := len(ciphertext)
	key := byte(0xAB)
	var nextKey byte
	for i := 0; i < n; i++ {
		nextKey = ciphertext[i]
		ciphertext[i] = ciphertext[i] ^ key
		key = nextKey
	}
	return string(ciphertext)
}

// sendUDP sends the command and does not wait for any response.
// Responses are handled by the listener thread.
func sendUDP(ip string, cmd string) error {
	if kasaUDPconn == nil {
		return fmt.Errorf("udp conn not running")
	}

	repeats := config.Get().KasaBroadcasts
	// unset, 0 (misconfigured) or 1 sends 1 packet
	if repeats < 1 {
		repeats = 1
	}

	payload := encrypt(cmd)
	for i := uint8(0); i < repeats; i++ {
		_, err := kasaUDPconn.WriteToUDP(payload, &net.UDPAddr{IP: net.ParseIP(ip), Port: 9999})
		if err != nil {
			log.Info.Printf("cannot send UDP command: %s", err.Error())
			return err
		}
	}
	return nil
}
