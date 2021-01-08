package kasa

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/platform"
	"net"
	"sync"
	"time"
)

// Platform is the platform handle for the Kasa stuff
type Platform struct {
	Running bool
}

// defined by kasa devices
type kasaDevice struct {
	System kasaSystem `json:"system"`
}

// defined by kasa devices
type kasaSystem struct {
	Sysinfo kasaSysinfo `json:"get_sysinfo"`
}

// defined by kasa devices
type kasaSysinfo struct {
	SWVersion  string `json:"sw_ver"`
	HWVersion  string `json:"hw_ver"`
	Model      string `json:"model"`
	DeviceID   string `json:"deviceId"`
	OEMID      string `json:"oemId"`
	HWID       string `json:"hwId"`
	RSSI       int    `json:"rssi"`
	Longitude  int    `json:"longitude_i"`
	Latitude   int    `json:"latitude_i"`
	Alias      string `json:"alias"`
	Status     string `json:"status"`
	MIC        string `json:"mic_type"`
	Feature    string `json:"feature"`
	MAC        string `json:"mac"`
	Updating   int    `json""updating"`
	LEDOff     int    `json:"led_off"`
	RelayState int    `json:"relay_state"`
	Brightness int    `json:"brightness"`
	OnTime     int    `json:"on_time"`
	ActiveMode string `json:"active_mode"`
	DevName    string `json:"dev_name"`
}

var kasas map[string]*tfaccessory.TFAccessory
var doOnce sync.Once

// Startup is called by the platform management to start the platform up
func (k Platform) Startup(c config.Config) platform.Control {
	k.Running = true
	go broadcastDiscovery()
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
		kasas = make(map[string]*tfaccessory.TFAccessory)
	})

	// pull switch to get a.Info -- override the config file with reality
	settings, err := getSettings(a)
	if err != nil {
		log.Info.Print("unable to identify kasa device, skipping: %s", err.Error())
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
	case "HS200(US)":
		a.Type = accessory.TypeSwitch
	case "HS220(US)":
		a.Type = accessory.TypeLightbulb
	default:
		a.Type = accessory.TypeSwitch
	}

	log.Info.Printf("adding [%s]: [%s]", a.Info.Name, a.Info.Model)
	// add to HC for GUI
	h, _ := platform.GetPlatform("HomeControl")
	h.AddAccessory(a)

	// kasas are indexed by IP address
	kasas[a.IP] = a

	if a.Switch != nil {
		// startup value
		a.Switch.Switch.On.SetValue(settings.RelayState > 0)

		// install callbacks: if we get an update from HC, deal with it
		a.Switch.Switch.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s] to [%t] from HC handler", a.Name, newstate)
			err := setRelayState(a, newstate)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			ks, err := getSettings(a)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			if (ks.RelayState > 0) != newstate {
				log.Info.Printf("unable to update kasa state to %t", newstate)
				a.Switch.Switch.On.SetValue(ks.RelayState > 0)
			}
		})
	}
	if a.HS220 != nil {
		a.HS220.Lightbulb.On.SetValue(settings.RelayState > 0)
		a.HS220.Lightbulb.ProgrammableSwitchOutputState.SetValue(settings.RelayState)
		a.HS220.Lightbulb.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s] to [%t] from HS220 handler", a.Name, newstate)
			err := setRelayState(a, newstate)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			ks, err := getSettings(a)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			if (ks.RelayState > 0) != newstate {
				log.Info.Printf("unable to update kasa state to %t", newstate)
				a.HS220.Lightbulb.On.SetValue(ks.RelayState > 0)
				a.HS220.Lightbulb.ProgrammableSwitchOutputState.SetValue(ks.RelayState)
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
			ks, err := getSettings(a)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			if ks.Brightness != newval {
				log.Info.Printf("unable to update kasa brightness to %d", newval)
				a.HS220.Lightbulb.Brightness.SetValue(ks.Brightness)
			}
		})
		a.HS220.Lightbulb.ProgrammableSwitchOutputState.OnValueRemoteUpdate(func(newval int) {
			log.Info.Printf("setting [%s] to [%d] from HS220 PSOS handler", a.Name, newval)
			err := setRelayState(a, newval == 1)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			ks, err := getSettings(a)
			if err != nil {
				log.Info.Println(err.Error())
				return
			}
			log.Info.Printf("%+v", ks)
		})
	}

	// actions
}

func setRelayState(a *tfaccessory.TFAccessory, newstate bool) error {
	// log.Info.Printf("setting kasa hardware state for [%s] to [%t]", a.Name, newstate)
	state := 0
	if newstate {
		state = 1
	}
	cmd := fmt.Sprintf(`{"system":{"set_relay_state":{"state":%d}}}`, state)
	_, err := send(a.IP, cmd)
	if err != nil {
		log.Info.Println(err.Error())
		return err
	}
	return nil
}

func setBrightness(a *tfaccessory.TFAccessory, newval int) error {
	cmd := fmt.Sprintf(`{"smartlife.iot.dimmer":{"set_brightness":{"brightness":%d}}}`, newval)
	_, err := send(a.IP, cmd)
	if err != nil {
		log.Info.Println(err.Error())
		return err
	}
	return nil
}

// GetAccessory looks up a Kasa device by IP address
func (k Platform) GetAccessory(ip string) (*tfaccessory.TFAccessory, bool) {
	val, ok := kasas[ip]
	return val, ok
}

func getSettings(a *tfaccessory.TFAccessory) (*kasaSysinfo, error) {
	// log.Info.Printf("full kasa pull for [%s]", a.Name)
	res, err := send(a.IP, `{"system":{"get_sysinfo":null}}`)
	if err != nil {
		log.Info.Println(err.Error())
		return nil, err
	}
	// log.Info.Println(res)

	var kd kasaDevice
	if err = json.Unmarshal([]byte(res), &kd); err != nil {
		log.Info.Println(err.Error())
		return nil, err
	}
	// log.Info.Printf("%+v", kd.System.Sysinfo)
	return &kd.System.Sysinfo, nil
}

// Background runs a background Go task verifying HC has the current state of the Kasa devices
func (k Platform) Background() {
	// check everything's status every minute
	go func() {
		for range time.Tick(time.Second * 60) {
			k.backgroundPuller()
		}
	}()
}

func (k Platform) backgroundPuller() {
	for _, a := range kasas {
		r, err := getSettings(a)
		if err != nil {
			log.Info.Println(err.Error())
			continue
		}
		// HS200
		if a.Switch != nil {
			if a.Switch.Switch.On.GetValue() != (r.RelayState > 0) {
				a.Switch.Switch.On.SetValue(r.RelayState > 0)
			}
		}
		// HS220
		if a.HS220 != nil {
			if a.HS220.Lightbulb.On.GetValue() != (r.RelayState > 0) {
				a.HS220.Lightbulb.On.SetValue(r.RelayState > 0)
				a.HS220.Lightbulb.ProgrammableSwitchOutputState.SetValue(r.RelayState)
			}
			if a.HS220.Lightbulb.Brightness.GetValue() != r.Brightness {
				a.HS220.Lightbulb.Brightness.SetValue(r.Brightness)
			}
		}
	}
}

func encrypt(plaintext string) []byte {
	n := len(plaintext)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(n))
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

func send(ip string, cmd string) (string, error) {
	payload := encrypt(cmd)
	r := net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: 9999,
	}

	conn, err := net.DialTCP("tcp4", nil, &r)
	if err != nil {
		log.Info.Printf("Cannot connnect to device: %s", err.Error())
		return "", err
	}
	_, err = conn.Write(payload)
	if err != nil {
		log.Info.Printf("Cannot send command to device: %s", err.Error())
		return "", err
	}

	// this is slow AF // data, err := ioutil.ReadAll(conn)
	// 200's return ~600 bytes, 220's return ~800 bytes; 1k should be enough
	// see go-eiscp's method for how to improve this
	data := make([]byte, 1024)
	n, err := conn.Read(data)
	if err != nil {
		log.Info.Println("Cannot read data from device:", err)
		return "", err
	}
	result := decrypt(data[4:n]) // start reading at 4, go to total bytes read
	return result, nil
}

func broadcastDiscovery() {
	// https://github.com/aler9/howto-udp-broadcast-golang
	listener, err := net.ListenPacket("udp4", ":9999")
	if err != nil {
		log.Info.Printf("unable to start discovery listener: %s", err.Error())
		return
	}

	// listen for 15 seconds for responses
	go func() {
		listener.SetReadDeadline(time.Now().Add(time.Second * 15))
		defer listener.Close()
		buf := make([]byte, 1024)
		for {
			n, responder, err := listener.ReadFrom(buf)
			if err != nil {
				log.Info.Printf("discovery listener: %s", err.Error())
				break
			}
			log.Info.Printf("kasa discovery: %s responded with: %s", responder, decrypt(buf[4:n]))
			// if it isn't already in the list of known devices, add it
		}
	}()

	// TODO: Get list of local subnets, scan each, don't hardcode my subnet
	payload := encrypt(`{"system":{"get_sysinfo":null},"smartlife.iot.dimmer":{"get_dimmer_parameters":null}}`)
	for i := 0; i < 3; i++ {
		log.Info.Println("subnet query")
		addr, err := net.ResolveUDPAddr("udp4", "192.168.1.255:9999")
		if err != nil {
			log.Info.Printf("discovery failed: %s", err.Error())
			return
		}
		_, err = listener.WriteTo(payload, addr)
		if err != nil {
			log.Info.Printf("discovery failed: %s", err.Error())
			return
		}

		log.Info.Println("broadcast query")
		addr, err = net.ResolveUDPAddr("udp4", "255.255.255.255:9999")
		if err != nil {
			log.Info.Printf("discovery failed: %s", err.Error())
			return
		}
		_, err = listener.WriteTo(payload, addr)
		if err != nil {
			log.Info.Printf("discovery failed: %s", err.Error())
			return
		}
		time.Sleep(time.Second * 3)
	}
}
