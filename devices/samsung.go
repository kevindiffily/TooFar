package devices

import (
	"github.com/McKael/samtv"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/service"
)

// tested w/ Samsung J-series 2015 TV
type SamsungTV struct {
	*accessory.Accessory

	SamTV      *samtv.SmartViewSession
	Television *SamsungTVSvc
	Speaker    *service.Speaker

	// added to Speaker
	VolumeActive *characteristic.Active
	Volume       *characteristic.Volume

	Sources map[uint8]string
}

func NewSamsungTV(info accessory.Info) *SamsungTV {
	acc := SamsungTV{}
	acc.Accessory = accessory.New(info, accessory.TypeTelevision)
	acc.Television = NewSamsungTVSvc()
	acc.Speaker = service.NewSpeaker()

	// when its off, it's off -- WOL doesn't seem to work either
	acc.Television.SleepDiscoveryMode.SetValue(characteristic.SleepDiscoveryModeNotDiscoverable)
	acc.Television.PowerModeSelection.SetValue(characteristic.PowerModeSelectionShow)
	acc.Television.Primary = true
	acc.AddService(acc.Television.Service)

	acc.Volume = characteristic.NewVolume()
	acc.Volume.Description = "Master Volume"
	acc.Speaker.AddCharacteristic(acc.Volume.Characteristic)

	acc.VolumeActive = characteristic.NewActive()
	acc.VolumeActive.Description = "Speaker Active"
	acc.VolumeActive.SetValue(characteristic.ActiveActive)
	acc.Volume.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("SamsungTV: HC requested speaker active: %d", newstate)
		acc.SamTV.Key("KEY_MUTE")
	})

	acc.Speaker.Mute.SetValue(false)
	acc.Speaker.Mute.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("SamsungTV: %t", newstate)
		acc.SamTV.Key("KEY_MUTE")
	})
	acc.Speaker.AddCharacteristic(acc.VolumeActive.Characteristic)
	acc.Speaker.Primary = false
	acc.AddService(acc.Speaker.Service)
	acc.Speaker.AddLinkedService(acc.Television.Service) // does not break

	return &acc
}

func (t *SamsungTV) AddInputs(sources map[uint8]string) {
	t.Sources = sources

	for id, name := range sources {
		log.Info.Printf("adding input source: %+v", name)
		is := service.NewInputSource()

		is.Name.SetValue(name)
		is.Name.Description = "Name"
		is.ConfiguredName.SetValue(name)
		is.ConfiguredName.Description = "ConfiguredName"
		inputSourceType := characteristic.InputSourceTypeHdmi
		inputDeviceType := characteristic.InputDeviceTypeAudioSystem
		is.InputSourceType.SetValue(inputSourceType)
		is.InputSourceType.Description = "InputSourceType"
		is.IsConfigured.SetValue(characteristic.IsConfiguredConfigured)
		is.IsConfigured.Description = "IsConfigured"
		is.CurrentVisibilityState.SetValue(characteristic.CurrentVisibilityStateShown)
		is.CurrentVisibilityState.Description = "CurrentVisibilityState"

		is.Identifier.SetValue(int(id))
		is.Identifier.Description = "Identifier"

		is.InputDeviceType.SetValue(inputDeviceType)
		is.InputDeviceType.Description = "InputDeviceType"
		is.TargetVisibilityState.SetValue(characteristic.TargetVisibilityStateHidden)
		is.TargetVisibilityState.Description = "TargetVisibilityState"

		// yes, both are required
		t.AddService(is.Service)
		t.Television.AddLinkedService(is.Service)

		is.TargetVisibilityState.OnValueRemoteUpdate(func(newstate int) {
			log.Info.Printf("%s TargetVisibilityState: %d", is.Name.GetValue(), newstate)
			is.CurrentVisibilityState.SetValue(newstate) // not saved, but fine for now
		})
		is.IsConfigured.OnValueRemoteUpdate(func(newstate int) {
			log.Info.Printf("%s IsConfigured: %d", is.Name.GetValue(), newstate)
		})
		is.Identifier.OnValueRemoteUpdate(func(newstate int) {
			log.Info.Printf("%s Identifier: %d", is.Name.GetValue(), newstate)
		})
	}
}

type SamsungTVSvc struct {
	*service.Service

	On                 *characteristic.On
	Volume             *characteristic.Volume
	StreamingStatus    *characteristic.StreamingStatus
	Active             *characteristic.Active
	ActiveIdentifier   *characteristic.ActiveIdentifier
	ConfiguredName     *characteristic.ConfiguredName
	SleepDiscoveryMode *characteristic.SleepDiscoveryMode
	Brightness         *characteristic.Brightness
	ClosedCaptions     *characteristic.ClosedCaptions
	DisplayOrder       *characteristic.DisplayOrder
	CurrentMediaState  *characteristic.CurrentMediaState
	TargetMediaState   *characteristic.TargetMediaState
	PictureMode        *characteristic.PictureMode
	PowerModeSelection *characteristic.PowerModeSelection
	RemoteKey          *characteristic.RemoteKey
}

func NewSamsungTVSvc() *SamsungTVSvc {
	svc := SamsungTVSvc{}
	svc.Service = service.New(service.TypeTelevision)

	svc.On = characteristic.NewOn()
	svc.AddCharacteristic(svc.On.Characteristic)
	svc.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("SamsungTV: HC requested On: %t", newstate)
	})

	svc.Volume = characteristic.NewVolume()
	svc.AddCharacteristic(svc.Volume.Characteristic)
	svc.Volume.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("SamsungTV: HC requested television volume: %d", newstate)
	})

	svc.StreamingStatus = characteristic.NewStreamingStatus()
	svc.AddCharacteristic(svc.StreamingStatus.Characteristic)
	svc.StreamingStatus.OnValueRemoteUpdate(func(newstate []byte) {
		log.Info.Printf("SamsungTV: HC requested StreamingStatus: %d", string(newstate))
	})

	svc.Active = characteristic.NewActive()
	svc.AddCharacteristic(svc.Active.Characteristic)
	svc.Active.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("SamsungTV: HC requested Active: %d", newstate)
	})

	svc.ActiveIdentifier = characteristic.NewActiveIdentifier()
	svc.AddCharacteristic(svc.ActiveIdentifier.Characteristic)
	svc.ActiveIdentifier.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("SamsungTV: HC requested ActiveIdentifier: %d", newstate)
	})

	svc.ConfiguredName = characteristic.NewConfiguredName()
	svc.AddCharacteristic(svc.ConfiguredName.Characteristic)
	svc.ActiveIdentifier.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("SamsungTV: HC requested ConfiguredName: %d", newstate)
	})

	svc.SleepDiscoveryMode = characteristic.NewSleepDiscoveryMode()
	svc.AddCharacteristic(svc.SleepDiscoveryMode.Characteristic)

	svc.Brightness = characteristic.NewBrightness()
	svc.AddCharacteristic(svc.Brightness.Characteristic)
	svc.Brightness.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("SamsungTV: HC requested Brightness: %d", newstate)
	})

	svc.ClosedCaptions = characteristic.NewClosedCaptions()
	svc.AddCharacteristic(svc.ClosedCaptions.Characteristic)
	svc.ClosedCaptions.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("SamsungTV: HC requested ClosedCaptions: %d", newstate)
	})

	svc.DisplayOrder = characteristic.NewDisplayOrder()
	svc.AddCharacteristic(svc.DisplayOrder.Characteristic)
	svc.DisplayOrder.OnValueRemoteUpdate(func(newstate []byte) {
		log.Info.Printf("SamsungTV: HC requested DisplayOrder: %s", string(newstate))
	})

	svc.CurrentMediaState = characteristic.NewCurrentMediaState()
	// svc.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePlay)
	svc.AddCharacteristic(svc.CurrentMediaState.Characteristic)

	svc.TargetMediaState = characteristic.NewTargetMediaState()
	svc.AddCharacteristic(svc.TargetMediaState.Characteristic)
	svc.TargetMediaState.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("SamsungTV: HC requested TargetMediaState: %d", newstate)
	})

	svc.PictureMode = characteristic.NewPictureMode()
	svc.AddCharacteristic(svc.PictureMode.Characteristic)
	svc.PictureMode.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("SamsungTV: HC requested PictureMode: %d", newstate)
	})

	svc.PowerModeSelection = characteristic.NewPowerModeSelection()
	svc.AddCharacteristic(svc.PowerModeSelection.Characteristic)
	svc.PowerModeSelection.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("SamsungTV: HC requested PowerModeSelection: %d", newstate)
		svc.PowerModeSelection.SetValue(newstate)
	})

	svc.RemoteKey = characteristic.NewRemoteKey()
	svc.AddCharacteristic(svc.RemoteKey.Characteristic)
	svc.RemoteKey.SetValue(characteristic.RemoteKeyInfo)

	return &svc
}
