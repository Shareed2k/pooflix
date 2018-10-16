package core

import (
	"github.com/hashicorp/mdns"
	"os"
)

func NewDns() error {
	// Setup our service export
	host, err := os.Hostname()
	if err != nil {
		return err
	}

	info := []string{"PooFlix mdns service"}
	service, _ := mdns.NewMDNSService(host, "_pooflix.local", "", "", 5353, nil, info)

	// Create the mDNS server, defer shutdown
	server, _ := mdns.NewServer(&mdns.Config{Zone: service})
	defer server.Shutdown()

	return nil
}
