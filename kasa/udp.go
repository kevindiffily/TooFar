package kasa

import (
	"encoding/json"
	"fmt"
	"github.com/brutella/hc/log"
	"github.com/cloudkucooland/toofar/platform"
	"strings"
)

func doUDPresponse(ip, res string) {
	k, ok := platform.GetPlatform("Kasa")
	if !ok {
		log.Info.Println("no kasa platform?")
		return
	}
	a, ok := k.GetAccessory(ip)
	if !ok {
		if res == cmd_sysinfo {
			// no need to log our own broadcasts
			return
		}
		log.Info.Printf("reply from unknown device: [%s] %s", ip, res)
		return
	}

	if strings.Contains(res, `"get_sysinfo"`) {
		var kd kasaDevice
		if err := json.Unmarshal([]byte(res), &kd); err != nil {
			log.Info.Println(err.Error())
			return
		}
		// log.Info.Printf("%+v", kd)
		r := kd.System.Sysinfo

		// HS200 / HS210
		if a.Switch != nil {
			if a.Switch.Switch.On.GetValue() != (r.RelayState > 0) {
				a.Switch.Switch.On.SetValue(r.RelayState > 0)
			}
		}

		// HS220
		if a.HS220 != nil {
			if a.HS220.Lightbulb.On.GetValue() != (r.RelayState > 0) {
				a.HS220.Lightbulb.On.SetValue(r.RelayState > 0)
				a.HS220.ProgrammableSwitch.ProgrammableSwitchOutputState.SetValue(r.RelayState)
			}
			if a.HS220.Lightbulb.Brightness.GetValue() != r.Brightness {
				a.HS220.Lightbulb.Brightness.SetValue(r.Brightness)
			}
		}
		return
	}

	if res == `{"system":{"set_relay_state":{"err_code":0}}}` {
		log.Info.Println("[%s] relay state changed", a.Name)
		return
	}

	if res == `{"smartlife.iot.dimmer":{"set_brightness":{"err_code":0}}}` {
		log.Info.Println("[%s] brightness changed", a.Name)
		return
	}

	fmt.Printf("unhandled kasa response [%s] %s\n", ip, res)
}
