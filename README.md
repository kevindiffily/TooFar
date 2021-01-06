TooFar : Another HomeKit bridge written in Go

This is glue to connect various other things together.

This is not really a general purpose bridge, it is what I need for my setup.

My goals:
    Apple's HomeKit (Home App) will be the primary UI; Siri will be second
    Don't use anything that requires a cloud connection other than the AppleTV.
    Almost all automation/configuration will take place in HomeKit, no vendor-specific automations.
    This bridge will support minimal automation, but ony where the same thing can't be achieved in HomeKit.
    Wherever possible, the "analog" switches must "do the right thing." If a switch is turned off, smart devices don't become unreachable.
    Small, fast, efficient

Constraints:
    Small investment in Ikea Tradfri
    3 Philips Hue can lights, connected to Tradfri--cannot be shown via HK since the vendors differ, need to bridge Tradfri
    TP-Link Kasa switches
    Shelly relays to make analog switches smart
   
To Do:
    Auto-discovery of Kasa switches, manual config is easy enough to make this low priority
    Auto-discovery of Shelly relays, manual config is easy enough to make this low priority
    Lightbulbs (e.g. the Philips Hue) that are not RGB, but cool/warm should not expose the RGB characteristics--no HC type for this, need to build my own
    Support Onkyo eISCP for my amps.
    Support Sony Blu-Ray player
    Support Samsung TV

What sucks:
    First setup of Tradfri requires go-tradfri client to get the username/password configured. One-time-problem
    Configuration requires reading my mind... or reading the code

