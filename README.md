# TooFar
Another HomeKit bridge written in Go

This is glue to connect various other things together.

This is not really a general purpose bridge, it is what I need for my setup. I'm making it public because that's what I do. If you find it useful, use it. 

# My goals for home automation:
* Analog (physical switches) is primary UI. Apple's HomeKit (Home App) will be the primary "non-analog" UI; Siri will be second; eschew vendor-specific apps wherever possible
* I've actually come to find asking siri to do things to be useful. Siri is becoming the primary UI.... until thread based motion sensors come out...
* Don't use anything that requires a cloud connection other than the AppleTV
* All automation/configuration will take place in HomeKit, no vendor-specific automations (e.g. don't use Kasa's scenes/smart actions)
* Wherever possible, the "analog" switches must "do the right thing." If a switch is turned off, smart devices don't become unreachable. Guests should not be confused or warned "don't use that switch."

# My goals for TooFar 
* Be small, fast, and efficient. Things like HomeBridge.io and openHAB are great, but too much for my needs. I found myself spending so much time bending them to my will that it just became easier to build my own. I think "small" has gone out the window at this point. But it is still smaller than other HK bridges.

# Constraints (the hardware that must work):
* Lots and lots of Apple stuff, homepod minis, macs, iPhones, Apple Watch....
* TP-Link Kasa switches and power strips (I like these. I have 30)
* Ikea Tradfri (I have a few bulbs, one hub) these do not need to be bridged ... but ...
* 3 Philips Hue can lights, connected to Tradfri. which cannot be shown via HK since the vendors differ, need to bridge Tradfri (or I could buy a Hue Hub...)
* Onkyo Receiver
* Konnected.io to use the stuff from the 1990's era hardwired alarm system

# Features
* Support for Onkyo/Pioneer/Integra amplifier/av-receivers by pretending to be a TV. Any eiscp Onkyo, Pioneer, or Integra AVR should work (including auto-detection of inputs)
* Support for OpenWeatherMap data -- you can automate other devices based on weather conditions using the "Controller" iOS app.

# To Do:
* Move a lot of stuff from the platform to the devices...

# What sucks:
* First setup of Tradfri requires go-tradfri client to get the username/password configured. One-time-problem
* Configuration requires reading my mind ... getting easier now that auto-discovery mostly works
* Onkyo eISCP is a 1980's serial protocol streaming over TCP, it gets WEIRD when a network stream is constantly updating the "now playing" info...
