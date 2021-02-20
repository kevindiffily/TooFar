package samsung

import (
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/devices"

	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
)

func handleRemote(a *tfaccessory.TFAccessory, newstate int) {
	d := a.Device.(*devices.SamsungTV).SamTV
	log.Info.Printf("samsung remote key: %d", newstate)
	switch newstate {
	case characteristic.RemoteKeyRewind:
		if err := d.Key("KEY_REWIND"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyFastForward:
		if err := d.Key("KEY_FF"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyExit:
		if err := d.Key("KEY_EXIT"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyPlayPause:
		if err := d.Key("KEY_PLAY"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyInfo:
		if err := d.Key("KEY_INFO"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyNextTrack:
		if err := d.Key("KEY_RIGHT"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyPrevTrack:
		if err := d.Key("KEY_LEFT"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyArrowUp:
		if err := d.Key("KEY_UP"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyArrowDown:
		if err := d.Key("KEY_DOWN"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyArrowLeft:
		if err := d.Key("KEY_LEFT"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyArrowRight:
		if err := d.Key("KEY_RIGHT"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeySelect:
		if err := d.Key("KEY_RETURN"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyBack:
		if err := d.Key("KEY_BACK_MHP"); err != nil {
			log.Info.Println(err)
		}
	}
}
