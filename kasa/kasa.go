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
	"github.com/cloudkucooland/toofar/platform"
	"net"
	"sync"
	"time"
)

// https://www.softscheck.com/en/reverse-engineering-tp-link-hs110/#TP-Link%20Smart%20Home%20Protocol
// https://medium.com/@hu3vjeen/reverse-engineering-tp-link-kc100-bac4641bf1cd
// https://machinekoder.com/controlling-tp-link-hs100110-smart-plugs-with-machinekit/
// https://lib.dr.iastate.edu/cgi/viewcontent.cgi?article=1424&context=creativecomponents
// https://github.com/p-doyle/Python-KasaSmartPowerStrip

const (
	cmd_sysinfo     = `{"system":{"get_sysinfo":{}}}`
	broadcast_sends = 1
)

var (
	broadcastIP = "192.168.1.255"
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
		return
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
	case "HS103(US)":
		a.Type = accessory.TypeSwitch
	case "HS200(US)":
		a.Type = accessory.TypeSwitch
	case "HS210(US)":
		a.Type = accessory.TypeSwitch
	case "HS220(US)":
		a.Type = accessory.TypeLightbulb
	case "KP303(US)":
		a.Type = accessory.TypeSwitch
	default:
		a.Type = accessory.TypeSwitch
	}

	log.Info.Printf("adding [%s]: [%s]", a.Info.Name, a.Info.Model)
	// add to HC for GUI
	hc.AddAccessory(a)

	// kasas are indexed by IP address
	kasas.mu.Lock()
	kasas.ks[a.IP] = a
	kasas.mu.Unlock()

	if a.Switch != nil {
		// startup value
		a.Switch.Switch.On.SetValue(settings.RelayState > 0)

		// install callbacks: if we get an update from HC, deal with it
		a.Switch.Switch.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s] to [%t] from Kasa switch handler", a.Name, newstate)
			err := setRelayState(a, newstate)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
		})
	}
	if a.HS220 != nil {
		a.HS220.Lightbulb.On.SetValue(settings.RelayState > 0)
		a.HS220.Lightbulb.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s] to [%t] from HS220 handler", a.Name, newstate)
			err := setRelayState(a, newstate)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
		})
		a.HS220.Lightbulb.Brightness.SetValue(settings.Brightness)
		a.HS220.Lightbulb.Brightness.OnValueRemoteUpdate(func(newval int) {
			log.Info.Printf("setting [%s] brightness [%d] from HS220 handler", a.Name, newval)
			err := setBrightness(a, newval)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
		})
	}
	if a.KP303 != nil {
		oneName := characteristic.NewName()
		oneName.SetValue(settings.Children[0].Alias)
		a.KP303.One.AddCharacteristic(oneName.Characteristic)

		a.KP303.One.On.SetValue(settings.Children[0].RelayState > 0)
		a.KP303.One.OutletInUse.SetValue(settings.Children[0].RelayState > 0)
		a.KP303.One.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s].[%s] to [%t] from KP303 handler", a.Name, settings.Children[0].Alias, newstate)
			err := setChildRelayState(a, settings.Children[0].ID, newstate)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
		})
		twoName := characteristic.NewName()
		twoName.SetValue(settings.Children[1].Alias)
		a.KP303.Two.AddCharacteristic(twoName.Characteristic)
		a.KP303.Two.On.SetValue(settings.Children[1].RelayState > 0)
		a.KP303.Two.OutletInUse.SetValue(settings.Children[1].RelayState > 0)
		a.KP303.Two.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s].[%s] to [%t] from KP303 handler", a.Name, settings.Children[1].Alias, newstate)
			err := setChildRelayState(a, settings.Children[1].ID, newstate)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
		})
		threeName := characteristic.NewName()
		threeName.SetValue(settings.Children[2].Alias)
		a.KP303.Three.AddCharacteristic(threeName.Characteristic)
		a.KP303.Three.On.SetValue(settings.Children[2].RelayState > 0)
		a.KP303.Three.OutletInUse.SetValue(settings.Children[2].RelayState > 0)
		a.KP303.Three.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s].[%s] to [%t] from KP303 handler", a.Name, settings.Children[2].Alias, newstate)
			err := setChildRelayState(a, settings.Children[2].ID, newstate)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
		})
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

func broadcastCmd(cmd string) error {
	// TODO look up proper broadcast addresses
	for i := 0; i < broadcast_sends; i++ {
		err := sendUDP(broadcastIP, cmd)
		if err != nil {
			log.Info.Println(err.Error())
			return err
		}
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

	payload := encrypt(cmd)
	_, err := kasaUDPconn.WriteToUDP(payload, &net.UDPAddr{IP: net.ParseIP(ip), Port: 9999})
	if err != nil {
		log.Info.Printf("cannot send UDP command: %s", err.Error())
		return err
	}
	return nil
}
