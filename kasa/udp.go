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

		// HS200 / HS210
		if a.Switch != nil {
			if a.Switch.Switch.On.GetValue() != (r.RelayState > 0) {
				log.Info.Printf("updating HomeKit: %s relay %d\n", a.IP, r.RelayState)
				a.Switch.Switch.On.SetValue(r.RelayState > 0)
			}
		}

		// HS220
		if a.HS220 != nil {
			if a.HS220.Lightbulb.On.GetValue() != (r.RelayState > 0) {
				log.Info.Printf("updating HomeKit: %s relay %d", a.IP, r.RelayState)
				a.HS220.Lightbulb.On.SetValue(r.RelayState > 0)
			}
			if a.HS220.Lightbulb.Brightness.GetValue() != r.Brightness {
				log.Info.Printf("updating HomeKit: %s brightness %d", a.IP, r.RelayState)
				a.HS220.Lightbulb.Brightness.SetValue(r.Brightness)
			}
		}
		return
	}

	// we should never see these, I was manually riggering them during testing
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
