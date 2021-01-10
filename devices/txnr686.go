package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/service"
	"strconv"
	"strings"

	"github.com/cloudkucooland/go-eiscp"
)

type TXNR686 struct {
	*accessory.Accessory
	Television *TXNR686Svc
	Speaker    *service.Speaker
	Temp       *service.TemperatureSensor

	// added to Speaker
	VolumeActive *characteristic.Active
	Volume       *characteristic.Volume

	// these break things if added
	VolumeControlType *characteristic.VolumeControlType
	VolumeSelector    *characteristic.VolumeSelector // bad things happen

	Sources map[int]string
}

func NewTXNR686(info accessory.Info) *TXNR686 {
	acc := TXNR686{}
	acc.Accessory = accessory.New(info, accessory.TypeTelevision)
	acc.Television = NewTXNR686Svc()
	acc.Speaker = service.NewSpeaker()
	acc.Temp = service.NewTemperatureSensor()

	acc.Television.SleepDiscoveryMode.SetValue(characteristic.SleepDiscoveryModeAlwaysDiscoverable)
	acc.Television.PowerModeSelection.SetValue(characteristic.PowerModeSelectionShow)
	acc.Television.Primary = true
	acc.AddService(acc.Television.Service)

	acc.Volume = characteristic.NewVolume()
	acc.Volume.Description = "Master Volume"
	acc.Speaker.AddCharacteristic(acc.Volume.Characteristic)

	acc.VolumeControlType = characteristic.NewVolumeControlType()
	acc.VolumeControlType.Description = "VolumeControlType"
	acc.VolumeControlType.SetValue(characteristic.VolumeControlTypeAbsolute)
	// this breaks things
	// acc.Speaker.AddCharacteristic(acc.VolumeControlType.Characteristic)
	acc.VolumeSelector = characteristic.NewVolumeSelector()
	acc.VolumeSelector.Description = "VolumeSelector"
	// this break things
	// acc.Speaker.AddCharacteristic(acc.VolumeSelector.Characteristic)

	acc.VolumeActive = characteristic.NewActive()
	acc.VolumeActive.Description = "Speaker Active"
	acc.VolumeActive.SetValue(characteristic.ActiveActive)
	acc.Speaker.AddCharacteristic(acc.VolumeActive.Characteristic)

	acc.Speaker.Mute.SetValue(false)
	acc.Speaker.Primary = false
	acc.AddService(acc.Speaker.Service)
	// this should be required but breaks things
	// acc.Television.AddLinkedService(acc.Speaker.Service) // XXX

	acc.Temp.Service.Primary = false
	acc.AddService(acc.Temp.Service)
	acc.Television.AddLinkedService(acc.Temp.Service)

	acc.Sources = make(map[int]string)

	return &acc
}

func (t *TXNR686) AddInputs(nfi *eiscp.NRI) {
	for _, s := range nfi.Device.SelectorList.Selector {
		// skip the label
		if s.ID == "80" {
			continue
		}
		log.Info.Printf("adding input source: %+v", s)
		is := service.NewInputSource()

		is.Name.SetValue(s.Name)
		is.Name.Description = "Name"
		is.ConfiguredName.SetValue(s.Name)
		is.ConfiguredName.Description = "ConfiguredName"
		inputSourceType := characteristic.InputSourceTypeHdmi
		inputDeviceType := characteristic.InputDeviceTypeAudioSystem
		switch strings.ToUpper(s.ID) {
		case eiscp.SrcCBL: // CBL/SAT
			inputSourceType = characteristic.InputSourceTypeOther
			inputDeviceType = characteristic.InputDeviceTypeAudioSystem
		case eiscp.SrcGame: // GAME
			inputSourceType = characteristic.InputSourceTypeHdmi
			inputDeviceType = characteristic.InputDeviceTypeTv
		case eiscp.SrcAux1: // AUX
			inputSourceType = characteristic.InputSourceTypeOther
			inputDeviceType = characteristic.InputDeviceTypeAudioSystem
		case eiscp.SrcPC: // PC
			inputSourceType = characteristic.InputSourceTypeOther
			inputDeviceType = characteristic.InputDeviceTypeAudioSystem
		case eiscp.SrcDVD: // BD/DVD
			inputSourceType = characteristic.InputSourceTypeHdmi
			inputDeviceType = characteristic.InputDeviceTypePlayback
		case eiscp.SrcStrm: // STRMBOX
			inputSourceType = characteristic.InputSourceTypeHdmi
			inputDeviceType = characteristic.InputDeviceTypeTv
		case eiscp.SrcTV: // TV
			inputSourceType = characteristic.InputSourceTypeHdmi
			inputDeviceType = characteristic.InputDeviceTypeTv
		case eiscp.SrcPhono: // Phono
			inputSourceType = characteristic.InputSourceTypeOther
			inputDeviceType = characteristic.InputDeviceTypeAudioSystem
		case eiscp.SrcCD: // CD
			inputSourceType = characteristic.InputSourceTypeOther
			inputDeviceType = characteristic.InputDeviceTypeAudioSystem
		case eiscp.SrcAM: // AM
			inputSourceType = characteristic.InputSourceTypeTuner
			inputDeviceType = characteristic.InputDeviceTypeTuner
		case eiscp.SrcFM: // FM
			inputSourceType = characteristic.InputSourceTypeTuner
			inputDeviceType = characteristic.InputDeviceTypeTuner
		case eiscp.SrcNetwork: // NET
			inputSourceType = characteristic.InputSourceTypeApplication
			inputDeviceType = characteristic.InputDeviceTypePlayback
		case eiscp.SrcBluetooth: // BLUETOOTH
			inputSourceType = characteristic.InputSourceTypeApplication
			inputDeviceType = characteristic.InputDeviceTypeAudioSystem
		}
		is.InputSourceType.SetValue(inputSourceType)
		is.InputSourceType.Description = "InputSourceType"
		is.IsConfigured.SetValue(characteristic.IsConfiguredConfigured)
		is.IsConfigured.Description = "IsConfigured"
		is.CurrentVisibilityState.SetValue(characteristic.CurrentVisibilityStateShown)
		is.CurrentVisibilityState.Description = "CurrentVisibilityState"

		// optional
		i, err := strconv.ParseInt(s.ID, 16, 32)
		if err != nil {
			log.Info.Println(err.Error())
		} else {
			is.Identifier.SetValue(int(i))
			is.Identifier.Description = "Identifier"
			t.Sources[int(i)] = s.Name
		}
		is.InputDeviceType.SetValue(inputDeviceType)
		is.InputDeviceType.Description = "InputDeviceType"
		is.TargetVisibilityState.SetValue(characteristic.TargetVisibilityStateHidden)
		is.TargetVisibilityState.Description = "TargetVisibilityState"

		// yes, both are required
		t.AddService(is.Service)
		t.Television.AddLinkedService(is.Service)

		// never triggered?
		is.CurrentVisibilityState.OnValueRemoteUpdate(func(newstate int) {
			log.Info.Printf("%s CurrentVisibilityState: %d", is.Name.GetValue(), newstate)
		})
		is.TargetVisibilityState.OnValueRemoteUpdate(func(newstate int) {
			log.Info.Printf("%s TargetVisibilityState: %d", is.Name.GetValue(), newstate)
			is.TargetVisibilityState.SetValue(newstate)  // not saved, but fine for now
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

type TXNR686Svc struct {
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

func NewTXNR686Svc() *TXNR686Svc {
	svc := TXNR686Svc{}
	svc.Service = service.New(service.TypeTelevision)

	svc.On = characteristic.NewOn()
	svc.AddCharacteristic(svc.On.Characteristic)
	svc.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("TXNR686: On: %t", newstate)
	})

	svc.Volume = characteristic.NewVolume()
	svc.AddCharacteristic(svc.Volume.Characteristic)
	svc.Volume.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("TXNR686: Volume: %d", newstate)
	})

	svc.StreamingStatus = characteristic.NewStreamingStatus()
	svc.AddCharacteristic(svc.StreamingStatus.Characteristic)
	svc.StreamingStatus.OnValueRemoteUpdate(func(newstate []byte) {
		log.Info.Printf("TXNR686: StreamingStatus: %d", string(newstate))
	})

	svc.Active = characteristic.NewActive()
	svc.AddCharacteristic(svc.Active.Characteristic)
	svc.Active.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("TXNR686: Requested Active be set to: %d", newstate)
	})

	svc.ActiveIdentifier = characteristic.NewActiveIdentifier()
	svc.AddCharacteristic(svc.ActiveIdentifier.Characteristic)
	svc.ActiveIdentifier.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("TXNR686: ActiveIdentifier: %d", newstate)
	})

	svc.ConfiguredName = characteristic.NewConfiguredName()
	svc.AddCharacteristic(svc.ConfiguredName.Characteristic)
	svc.ActiveIdentifier.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("TXNR686: ConfiguredName: %d", newstate)
	})

	svc.SleepDiscoveryMode = characteristic.NewSleepDiscoveryMode()
	svc.AddCharacteristic(svc.SleepDiscoveryMode.Characteristic)

	svc.Brightness = characteristic.NewBrightness()
	svc.AddCharacteristic(svc.Brightness.Characteristic)
	svc.Brightness.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("TXNR686: Brightness: %d", newstate)
	})

	svc.ClosedCaptions = characteristic.NewClosedCaptions()
	svc.AddCharacteristic(svc.ClosedCaptions.Characteristic)
	svc.ClosedCaptions.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("TXNR686: ClosedCaptions: %d", newstate)
	})

	svc.DisplayOrder = characteristic.NewDisplayOrder()
	svc.AddCharacteristic(svc.DisplayOrder.Characteristic)
	svc.DisplayOrder.OnValueRemoteUpdate(func(newstate []byte) {
		log.Info.Printf("TXNR686: DisplayOrder: %s", string(newstate))
	})

	svc.CurrentMediaState = characteristic.NewCurrentMediaState()
	svc.AddCharacteristic(svc.CurrentMediaState.Characteristic)
	svc.CurrentMediaState.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("TXNR686: CurrentMediaState: %d", newstate)
	})

	svc.TargetMediaState = characteristic.NewTargetMediaState()
	svc.AddCharacteristic(svc.TargetMediaState.Characteristic)
	svc.TargetMediaState.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("TXNR686: TargetMediaState: %d", newstate)
	})

	svc.PictureMode = characteristic.NewPictureMode()
	svc.AddCharacteristic(svc.PictureMode.Characteristic)
	svc.PictureMode.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("TXNR686: PictureMode: %d", newstate)
	})

	svc.PowerModeSelection = characteristic.NewPowerModeSelection()
	svc.AddCharacteristic(svc.PowerModeSelection.Characteristic)
	svc.PowerModeSelection.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("TXNR686: PowerModeSelection: %d", newstate)
		svc.PowerModeSelection.SetValue(newstate)
	})

	svc.RemoteKey = characteristic.NewRemoteKey()
	svc.AddCharacteristic(svc.RemoteKey.Characteristic)
	svc.RemoteKey.SetValue(characteristic.RemoteKeyInfo)

	return &svc
}

/*
func (t *TXNR686) UpdateTemp() {
	cent, err := dev.GetTempData()
	if err != nil {
		log.Info.Println(err.Error())
	}
	cint, err := strconv.Atoi(cent)
	if err != nil {
		cint = 20
	}
	t.Temp.CurrentTemperature.SetValue(float64(cint))
} */
