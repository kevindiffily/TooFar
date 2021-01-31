package devices

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/service"
	"github.com/cloudkucooland/go-eiscp"
	"strconv"
)

type OnkyoController struct {
	*accessory.Accessory

	Television *service.Television
	Speaker    *service.Speaker

	MusicOptimizer *ToggleSvc
	Volume         *FaderSvc
	Dimmer         *FaderSvc
	Parent         interface{}
	LMDs           map[int]string
}

func NewOnkyoController(info accessory.Info) *OnkyoController {
	acc := OnkyoController{}
	acc.Accessory = accessory.New(info, accessory.TypeTelevision)
	acc.Television = service.NewTelevision()
	acc.Speaker = service.NewSpeaker()

	vol := characteristic.NewVolume()
	vol.Description = "Unused"
	acc.Speaker.AddCharacteristic(vol.Characteristic)

	vct := characteristic.NewVolumeControlType()
	// vct.Description = "VolumeControlType"
	vct.SetValue(characteristic.VolumeControlTypeAbsolute)
	acc.Speaker.AddCharacteristic(vct.Characteristic)

	va := characteristic.NewActive()
	// va.Description = "Speaker Active"
	va.SetValue(characteristic.ActiveActive)
	acc.Speaker.AddCharacteristic(va.Characteristic)
	acc.Speaker.Mute.SetValue(false)
	acc.Speaker.Primary = false
	acc.AddService(acc.Speaker.Service)
	acc.Speaker.AddLinkedService(acc.Television.Service)
	acc.Speaker.Primary = false

	acc.Television.Primary = true
	acc.Television.SleepDiscoveryMode.SetValue(characteristic.SleepDiscoveryModeAlwaysDiscoverable)
	acc.Television.PowerModeSelection.SetValue(characteristic.PowerModeSelectionShow)
	acc.AddService(acc.Television.Service)

	acc.MusicOptimizer = NewToggleSvc("Music Optimizer")
	acc.MusicOptimizer.Primary = false
	acc.AddService(acc.MusicOptimizer.Service)

	acc.Dimmer = NewFaderSvc("Dimmer")
	acc.Dimmer.Primary = false
	acc.Dimmer.Value.SetMinValue(0)
	acc.Dimmer.Value.SetMaxValue(99)
	acc.Dimmer.Value.StepValue = 33
	acc.Dimmer.Value.SetValue(99)
	acc.AddService(acc.Dimmer.Service)

	acc.Volume = NewFaderSvc("Volume")
	acc.Volume.Primary = false
	acc.Volume.Value.SetMinValue(40)
	acc.Volume.Value.SetMaxValue(65)
	acc.Volume.Value.SetValue(55)
	acc.AddService(acc.Volume.Service)

	acc.LMDs = make(map[int]string)
	acc.AddLMD()
	return &acc
}

type ToggleSvc struct {
	*service.Service

	On   *characteristic.On
	Name *characteristic.Name
}

func NewToggleSvc(name string) *ToggleSvc {
	svc := ToggleSvc{}
	svc.Service = service.New(service.TypeSwitch)

	svc.On = characteristic.NewOn()
	svc.AddCharacteristic(svc.On.Characteristic)

	svc.Name = characteristic.NewName()
	svc.Name.SetValue(name)
	svc.AddCharacteristic(svc.Name.Characteristic)

	return &svc
}

type FaderSvc struct {
	*service.Service

	Value  *characteristic.Brightness
	Name   *characteristic.Name
	Active *characteristic.Active
}

func NewFaderSvc(name string) *FaderSvc {
	svc := FaderSvc{}
	svc.Service = service.New(service.TypeLightbulb)

	svc.Value = characteristic.NewBrightness()
	svc.AddCharacteristic(svc.Value.Characteristic)

	svc.Name = characteristic.NewName()
	svc.Name.SetValue(name)
	svc.AddCharacteristic(svc.Name.Characteristic)

	svc.Active = characteristic.NewActive()
	svc.Active.SetValue(characteristic.ActiveActive)
	svc.AddCharacteristic(svc.Active.Characteristic)

	return &svc
}

func (t *OnkyoController) AddLMD() {
	for k, v := range eiscp.ListeningModes {
		// skip the label
		log.Info.Printf("adding listening mode: %+v", v)
		l := service.NewInputSource()

		l.Name.SetValue(v)
		l.Name.Description = "Name"
		l.ConfiguredName.SetValue(v)
		l.ConfiguredName.Description = "ConfiguredName"
		l.InputSourceType.SetValue(characteristic.InputSourceTypeHdmi)
		l.InputSourceType.Description = "InputSourceType"
		l.IsConfigured.SetValue(characteristic.IsConfiguredConfigured)
		l.IsConfigured.Description = "IsConfigured"
		l.CurrentVisibilityState.SetValue(characteristic.CurrentVisibilityStateShown)
		l.CurrentVisibilityState.Description = "CurrentVisibilityState"

		// optional
		i, err := strconv.ParseInt(k, 16, 32)
		if err != nil {
			log.Info.Println(err.Error())
		} else {
			l.Identifier.SetValue(int(i))
			l.Identifier.Description = "Identifier"
			t.LMDs[int(i)] = v
		}
		l.InputDeviceType.SetValue(characteristic.InputDeviceTypeAudioSystem)
		l.InputDeviceType.Description = "InputDeviceType"
		l.TargetVisibilityState.SetValue(characteristic.TargetVisibilityStateHidden)
		l.TargetVisibilityState.Description = "TargetVisibilityState"

		// yes, both are required
		t.AddService(l.Service)
		t.Television.AddLinkedService(l.Service)

		// never triggered?
		l.CurrentVisibilityState.OnValueRemoteUpdate(func(newstate int) {
			log.Info.Printf("%s CurrentVisibilityState: %d", l.Name.GetValue(), newstate)
		})
		l.TargetVisibilityState.OnValueRemoteUpdate(func(newstate int) {
			log.Info.Printf("%s TargetVisibilityState: %d", l.Name.GetValue(), newstate)
			l.CurrentVisibilityState.SetValue(newstate) // not saved, but fine for now
		})
		l.IsConfigured.OnValueRemoteUpdate(func(newstate int) {
			log.Info.Printf("%s IsConfigured: %d", l.Name.GetValue(), newstate)
		})
		l.Identifier.OnValueRemoteUpdate(func(newstate int) {
			log.Info.Printf("%s Identifier: %d", l.Name.GetValue(), newstate)
		})
		l.ConfiguredName.OnValueRemoteUpdate(func(newname string) {
			log.Info.Printf("changing input name [%s]: %s", k, newname)
			// _, err := t.Parent.Amp.SetGetOne("IRN", fmt.Sprintf("%s%s", k, newname))
			if err != nil {
				log.Info.Printf(err.Error())
			}
		})
	}
	active := characteristic.NewActive()
	t.Television.AddCharacteristic(active.Characteristic)
	active.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("onkyo-controller: HC requested Active: %d", newstate)
	})

	ai := characteristic.NewActiveIdentifier()
	t.Television.AddCharacteristic(ai.Characteristic)
	ai.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("onkyo-controller: HC requested ActiveIdentifier: %d", newstate)
	})
}
