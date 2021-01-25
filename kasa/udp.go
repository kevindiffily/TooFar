package kasa

import (
	"encoding/json"
	"fmt"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/log"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/devices"
	"github.com/cloudkucooland/toofar/platform"
	"strings"
)

func doUDPresponse(ip, res string) {
	if res == cmd_sysinfo {
		// no need to log our own broadcasts
		return
	}

	k, ok := platform.GetPlatform("Kasa")
	if !ok {
		log.Info.Println("no kasa platform?")
		return
	}
	a, ok := k.GetAccessory(ip)
	if !ok {
		if !config.Get().Discover {
			return
		}
		log.Info.Printf("adding previously unknown device: %s", ip)
		newAcc := &tfaccessory.TFAccessory{Platform: "Kasa", IP: ip, Name: ip}
		k.AddAccessory(newAcc)
		return
	}

	if strings.Contains(res, `"get_sysinfo"`) {
		kd := kasaDevice{}
		if err := json.Unmarshal([]byte(res), &kd); err != nil {
			log.Info.Println(err.Error())
			return
		}
		// log.Info.Printf("%+v", kd)
		r := kd.System.Sysinfo

		switch a.Device.(type) {
		case *accessory.Switch:
			sw := a.Device.(*accessory.Switch)
			if sw.Switch.On.GetValue() != (r.RelayState > 0) {
				log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d\n", a.IP, r.Alias, r.RelayState)
				sw.Switch.On.SetValue(r.RelayState > 0)
			}
		case *devices.HS220:
			hs := a.Device.(devices.HS220)
			if hs.Lightbulb.On.GetValue() != (r.RelayState > 0) {
				log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d", a.IP, r.Alias, r.RelayState)
				hs.Lightbulb.On.SetValue(r.RelayState > 0)
			}
			if hs.Lightbulb.Brightness.GetValue() != r.Brightness {
				log.Info.Printf("updating HomeKit: [%s]:[%s] brightness %d", a.IP, r.Alias, r.RelayState)
				hs.Lightbulb.Brightness.SetValue(r.Brightness)
			}
		case *devices.KP303:
			kp := a.Device.(devices.KP303)
			for i := 0; i < len(kp.Outlets); i++ {
				if kp.Outlets[i].On.GetValue() != (r.Children[i].RelayState > 0) {
					log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d", a.IP, r.Children[i].Alias, r.Children[i].RelayState)
					kp.Outlets[i].On.SetValue(r.Children[0].RelayState > 0)
					kp.Outlets[i].OutletInUse.SetValue(r.Children[0].RelayState > 0)
				}
			}
		}
		return
	}

	/* if res == `{"system":{"set_relay_state":{"err_code":0}}}` {
		log.Info.Printf("[%s] relay state changed", a.Name)
		return
	}

	if res == `{"smartlife.iot.dimmer":{"set_brightness":{"err_code":0}}}` {
		log.Info.Printf("[%s] brightness changed", a.Name)
		return
	} */

	fmt.Printf("unhandled kasa response [%s] %s", ip, res)
}
