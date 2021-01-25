# TooFar
Another HomeKit bridge written in Go

This is glue to connect various other things together.

This is not really a general purpose bridge, it is what I need for my setup. I'm making it public because that's what I do. If you find it useful, use it. 

# My goals for home automation:
* Analog (physical switches) is primary UI. Apple's HomeKit (Home App) will be the primary "non-analog" UI; Siri will be second; eschew vendor-specific apps wherever possible
* Don't use anything that requires a cloud connection other than the AppleTV
* Almost all automation/configuration will take place in HomeKit, no vendor-specific automations (e.g. don't use Kasa's scenes/smart actions)
* Wherever possible, the "analog" switches must "do the right thing." If a switch is turned off, smart devices don't become unreachable. Guests should not be confused or warned "don't use that switch."

# My goals for TooFar 
* TooFar will support minimal automation, but ony where the same thing can't be achieved in HomeKit. Automate with homekit where possible, if not possible then write the automation bits in TooFar.
* Be small, fast, and efficient. Things like HomeBridge.io and openHAB are great, but too much for my needs. I found myself spending so much time bending them to my will that it just became easier to build my own

# Constraints (the hardware that must work):
* TP-Link Kasa switches (I like these. I have a dozen)
* Shelly relays to make analog switches smart (I have two, they are excellent, but more work than the Kasa)
* Ikea Tradfri (I have a few bulbs, one hub) these do not need to be bridged ... but ...
* 3 Philips Hue can lights, connected to Tradfri. which cannot be shown via HK since the vendors differ, need to bridge Tradfri (or I could buy a Hue Hub...)

# Features
* Support for Onkyo/Pioneer/Integra amplifier/av-receivers by pretending to be a TV. They are listed as TX-NR686, but any eiscp Onkyo, Pioneer or Integra AVR should work (including auto-detection of inputs)
* Support for checking if devices are up (http-ping)
* Support for OpenWeatherMap data -- you can automate other devices based on weather conditions using the "Controller" iOS app.

# To Do:
* Auto-discovery of Kasa switches (99%)
* Auto-discovery of Shelly relays, manual config is easy enough to make this low priority
* Support Sony Blu-Ray player -- not really a priority since HDMI-CEC gets the job done
* Support Samsung TV -- not really a priority since HDMI-CEC gets the job done
* Keep improving my fork of go-eiscp and integration of Onkyo amps with homekit

# What sucks:
* First setup of Tradfri requires go-tradfri client to get the username/password configured. One-time-problem
* Configuration requires reading my mind ... or reading the code ... 
* Actions are very basic at the moment. I'm only writing what I need/want
* Onkyo eISCP is a 1980's serial protocol streaming over TCP, it gets WEIRD when a network stream is constantly updating the "now playing" info...
