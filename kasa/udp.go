package kasa

import (
	"encoding/json"
	"fmt"
	"github.com/brutella/hc/characteristic"
	// "github.com/brutella/hc/accessory"
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
		if _, ok := kasas.ignore[ip]; ok {
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
		case *devices.KP115:
			sw := a.Device.(*devices.KP115)
			if sw.Outlet.On.GetValue() != (r.RelayState > 0) {
				log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d\n", a.IP, r.Alias, r.RelayState)
				sw.Outlet.On.SetValue(r.RelayState > 0)
				sw.Outlet.OutletInUse.SetValue(r.RelayState > 0)
			}
		case *devices.HS103:
			sw := a.Device.(*devices.HS103)
			if sw.Outlet.On.GetValue() != (r.RelayState > 0) {
				log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d\n", a.IP, r.Alias, r.RelayState)
				sw.Outlet.On.SetValue(r.RelayState > 0)
				sw.Outlet.OutletInUse.SetValue(r.RelayState > 0)
			}
		case *devices.HS200: // and 210
			sw := a.Device.(*devices.HS200)
			if sw.Switch.On.GetValue() != (r.RelayState > 0) {
				log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d\n", a.IP, r.Alias, r.RelayState)
				sw.Switch.On.SetValue(r.RelayState > 0)
			}
		case *devices.HS220:
			hs := a.Device.(*devices.HS220)
			if hs.Lightbulb.On.GetValue() != (r.RelayState > 0) {
				log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d", a.IP, r.Alias, r.RelayState)
				hs.Lightbulb.On.SetValue(r.RelayState > 0)
			}
			if hs.Lightbulb.Brightness.GetValue() != r.Brightness {
				log.Info.Printf("updating HomeKit: [%s]:[%s] brightness %d", a.IP, r.Alias, r.RelayState)
				hs.Lightbulb.Brightness.SetValue(r.Brightness)
			}
		case *devices.KP303:
			kp := a.Device.(*devices.KP303)
			for i := 0; i < len(kp.Outlets); i++ {
				if kp.Outlets[i].On.GetValue() != (r.Children[i].RelayState > 0) {
					log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d", a.IP, r.Children[i].Alias, r.Children[i].RelayState)
					kp.Outlets[i].On.SetValue(r.Children[0].RelayState > 0)
					kp.Outlets[i].OutletInUse.SetValue(r.Children[0].RelayState > 0)
				}
			}
		default:
			log.Info.Printf("unhandled sysinfo: %s\n", res)
		}
		return
	}

	if strings.Contains(res, `"count_down"`) {
		// log.Info.Printf("processing: %s\n", res)
		kd := kasaDevice{}
		if err := json.Unmarshal([]byte(res), &kd); err != nil {
			log.Info.Println(err.Error())
			return
		}
		if strings.Contains(res, `"get_rules"`) {
			var hasRules bool
			var remaining int
			if len(kd.Countdown.GetRules.RuleList) != 0 {
				hasRules = true
			}
			// if none of the countdowns are active, purge them all
			if hasRules {
				var activeRules bool
				for _, r := range kd.Countdown.GetRules.RuleList {
					if r.Enable != 0 {
						activeRules = true
						remaining = int(r.Remaining)
						break
					}
				}
				if !activeRules {
					deleteCountdown(a)
					hasRules = false
				}
			}
			// if it doesn't have rules, make sure it is in No-Program mode, else make sure it is is Program mode
			switch a.Device.(type) {
			case *devices.KP115:
				kp := a.Device.(*devices.KP115)
				ps := kp.Outlet.ProgramMode.GetValue()
				if !hasRules && ps == characteristic.ProgramModeProgramScheduled {
					kp.Outlet.ProgramMode.SetValue(characteristic.ProgramModeNoProgramScheduled)
				}
				if hasRules && ps == characteristic.ProgramModeNoProgramScheduled {
					kp.Outlet.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
					kp.Outlet.SetDuration.SetValue(0)
				}
				if remaining != 0 {
					kp.Outlet.RemainingDuration.SetValue(remaining)
				}
			case *devices.KP303:
				// ignore for now
			case *devices.HS103:
				kp := a.Device.(*devices.HS103)
				ps := kp.Outlet.ProgramMode.GetValue()
				if !hasRules && ps == characteristic.ProgramModeProgramScheduled {
					kp.Outlet.ProgramMode.SetValue(characteristic.ProgramModeNoProgramScheduled)
				}
				if hasRules && ps == characteristic.ProgramModeNoProgramScheduled {
					kp.Outlet.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
					kp.Outlet.SetDuration.SetValue(0)
				}
				if remaining != 0 {
					kp.Outlet.RemainingDuration.SetValue(remaining)
				}
			case *devices.HS200:
				kp := a.Device.(*devices.HS200)
				ps := kp.Switch.ProgramMode.GetValue()
				if !hasRules && ps == characteristic.ProgramModeProgramScheduled {
					kp.Switch.ProgramMode.SetValue(characteristic.ProgramModeNoProgramScheduled)
				}
				if hasRules && ps == characteristic.ProgramModeNoProgramScheduled {
					kp.Switch.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
					kp.Switch.SetDuration.SetValue(0)
				}
				if remaining != 0 {
					kp.Switch.RemainingDuration.SetValue(remaining)
				}
			case *devices.HS220:
				kp := a.Device.(*devices.HS220)
				ps := kp.Lightbulb.ProgramMode.GetValue()
				if !hasRules && ps == characteristic.ProgramModeProgramScheduled {
					kp.Lightbulb.ProgramMode.SetValue(characteristic.ProgramModeNoProgramScheduled)
				}
				if hasRules && ps == characteristic.ProgramModeNoProgramScheduled {
					kp.Lightbulb.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
					kp.Lightbulb.SetDuration.SetValue(0)
				}
				if remaining != 0 {
					kp.Lightbulb.RemainingDuration.SetValue(remaining)
				}
			default:
				log.Info.Printf("unhandled countdown.getrules: %s\n", res)
			}
			return
		}

		if strings.Contains(res, "add_rule") {
			// remaining := int(kd.Countdown.GetRules.RuleList[0].Remaining)
			switch a.Device.(type) {
			case *devices.KP115:
				hs := a.Device.(*devices.KP115)
				hs.Outlet.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
				// hs.Outlet.RemainingDuration.SetValue(remaining)
			case *devices.KP303:
				// ignore
			case *devices.HS103:
				hs := a.Device.(*devices.HS103)
				hs.Outlet.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
				// hs.Outlet.RemainingDuration.SetValue(remaining)
			case *devices.HS200:
				hs := a.Device.(*devices.HS200)
				hs.Switch.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
				// hs.Switch.RemainingDuration.SetValue(remaining)
			case *devices.HS220:
				hs := a.Device.(*devices.HS220)
				hs.Lightbulb.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
				// hs.Lightbulb.RemainingDuration.SetValue(remaining)
			default:
				log.Info.Printf("unhandled countdown.addrule: %s\n", res)
			}
			return
		}

		if strings.Contains(res, "delete_all_rules") {
			switch a.Device.(type) {
			case *devices.KP115:
				hs := a.Device.(*devices.KP115)
				hs.Outlet.ProgramMode.SetValue(characteristic.ProgramModeNoProgramScheduled)
				hs.Outlet.RemainingDuration.SetValue(0)
				hs.Outlet.SetDuration.SetValue(0)
			case *devices.KP303:
				// ignore
			case *devices.HS103:
				hs := a.Device.(*devices.HS103)
				hs.Outlet.ProgramMode.SetValue(characteristic.ProgramModeNoProgramScheduled)
				hs.Outlet.RemainingDuration.SetValue(0)
				hs.Outlet.SetDuration.SetValue(0)
			case *devices.HS200:
				hs := a.Device.(*devices.HS200)
				hs.Switch.ProgramMode.SetValue(characteristic.ProgramModeNoProgramScheduled)
				hs.Switch.RemainingDuration.SetValue(0)
				hs.Switch.SetDuration.SetValue(0)
			case *devices.HS220:
				hs := a.Device.(*devices.HS220)
				hs.Lightbulb.ProgramMode.SetValue(characteristic.ProgramModeNoProgramScheduled)
				hs.Lightbulb.RemainingDuration.SetValue(0)
				hs.Lightbulb.SetDuration.SetValue(0)
			default:
				log.Info.Printf("unhandled countdown.delrules: %s\n", res)
			}
			return
		}

	}

	if res == `{"system":{"set_relay_state":{"err_code":0}}}` {
		// log.Info.Printf("[%s] relay state changed", a.Name)
		return
	}

	if res == `{"smartlife.iot.dimmer":{"set_brightness":{"err_code":0}}}` {
		// log.Info.Printf("[%s] brightness changed", a.Name)
		return
	}

	fmt.Printf("unhandled kasa response [%s] %s\n", ip, res)
}
