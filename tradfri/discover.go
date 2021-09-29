package tradfri

import (
	"context"
	"strings"
	"time"

	"github.com/brutella/dnssd"
	"github.com/brutella/hc/log"
)

func discover() (string, error) {
	discovered := ""
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	found := func(e dnssd.BrowseEntry) {
		if strings.Contains(e.Name, "gw-") {
			// look through the list of IPs, pick something IPv4, IPV6 doesn't seem to work
			for _, ipa := range e.IPs {
				if ipa.To4() != nil {
					discovered = ipa.String()
					cancel()
					return
				}
			}
		}
	}

	if err := dnssd.LookupType(ctx, "_coap._udp.local.", found, reject); err != nil {
		if err.Error() != "context canceled" {
			log.Info.Printf("tradfri discovery: %v\n", err)
			return discovered, err
		}
	}
	return discovered, nil
}

func reject(e dnssd.BrowseEntry) {
	log.Info.Printf("dnssd-lookup: %+v", e)
}
