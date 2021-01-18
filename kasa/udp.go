package kasa

import (
	"encoding/json"
	"fmt"
	"github.com/brutella/hc/log"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
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

		if a.Switch != nil {
			if a.Switch.Switch.On.GetValue() != (r.RelayState > 0) {
				log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d\n", a.IP, r.Alias, r.RelayState)
				a.Switch.Switch.On.SetValue(r.RelayState > 0)
			}
		}

		if a.HS220 != nil {
			if a.HS220.Lightbulb.On.GetValue() != (r.RelayState > 0) {
				log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d", a.IP, r.Alias, r.RelayState)
				a.HS220.Lightbulb.On.SetValue(r.RelayState > 0)
			}
			if a.HS220.Lightbulb.Brightness.GetValue() != r.Brightness {
				log.Info.Printf("updating HomeKit: [%s]:[%s] brightness %d", a.IP, r.Alias, r.RelayState)
				a.HS220.Lightbulb.Brightness.SetValue(r.Brightness)
			}
		}

		if a.KP303 != nil {
			for i := 0; i <= 2; i++ {
				if a.KP303.Outlets[i].On.GetValue() != (r.Children[i].RelayState > 0) {
					log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d", a.IP, r.Children[i].Alias, r.Children[i].RelayState)
					a.KP303.Outlets[i].On.SetValue(r.Children[0].RelayState > 0)
					a.KP303.Outlets[i].OutletInUse.SetValue(r.Children[0].RelayState > 0)
				}
			}
		}
		return
	}

	if res == `{"system":{"set_relay_state":{"err_code":0}}}` {
		log.Info.Printf("[%s] relay state changed", a.Name)
		return
	}

	if res == `{"smartlife.iot.dimmer":{"set_brightness":{"err_code":0}}}` {
		log.Info.Printf("[%s] brightness changed", a.Name)
		return
	}

	fmt.Printf("unhandled kasa response [%s] %s", ip, res)
}
