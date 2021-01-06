# TooFar
Another HomeKit bridge written in Go

This is glue to connect various other things together.

This is not really a general purpose bridge, it is what I need for my setup. I'm making it public because that's what I do. If you find it useful, use it. 

# My goals:
    *Apple's HomeKit (Home App) will be the primary UI; Siri will be second
    *Don't use anything that requires a cloud connection other than the AppleTV.
    *Almost all automation/configuration will take place in HomeKit, no vendor-specific automations.
    *This bridge will support minimal automation, but ony where the same thing can't be achieved in HomeKit.
    *Wherever possible, the "analog" switches must "do the right thing." If a switch is turned off, smart devices don't become unreachable.
    *Small, fast, efficient

# Constraints (what I have to contend with):
    *Small investment in Ikea Tradfri (a few bulbs, one hub)
    *3 Philips Hue can lights, connected to Tradfri--which cannot be shown via HK since the vendors differ, need to bridge Tradfri
    *TP-Link Kasa switches (I like these. I have a dozen)
    *Shelly relays to make analog switches smart (I have two, they are excellent, but more work than the Kasa)
   
# To Do:
    *Auto-discovery of Kasa switches, manual config is easy enough to make this low priority
    *Auto-discovery of Shelly relays, manual config is easy enough to make this low priority
    *Support Sony Blu-Ray player -- not really a priority since HDMI-CEC gets the job done
    *Support Samsung TV -- not really a priority since HDMI-CEC gets the job done

# What sucks:
    *First setup of Tradfri requires go-tradfri client to get the username/password configured. One-time-problem
    *Configuration requires reading my mind... or reading the code
    *Actions are very basic at the moment. I'm only writing what I need/want
    *Onkyo eISCP needs real documentation. I've reversed-engineered a bit and will do more as I feel like it.
