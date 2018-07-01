package nat

import (
	"github.com/NebulousLabs/go-upnp"
)

// LocalPortMappingInfo denotes a single port being forwarde on the
// UPnP interface.
type LocalPortMappingInfo struct {
	LocalPort      uint16
	ExternalPort   uint16
	ExternalIP     string
	RouterLocation string
}

// Close clears a port from remaining open.
func (m *LocalPortMappingInfo) Close() {
	gateway, err := upnp.Load(m.RouterLocation)
	if err == nil {
		gateway.Clear(m.ExternalPort)
	}
}

// ForwardPort accesses the UPnP interface (should it be available) and port-forwards
// a specified local port.
func ForwardPort(localPort uint16) (*LocalPortMappingInfo, error) {
	gateway, err := upnp.Discover()
	if err != nil {
		return nil, err
	}

	ip, err := gateway.ExternalIP()
	if err != nil {
		return nil, err
	}

	err = gateway.Forward(localPort, "Noise")
	if err != nil {
		return nil, err
	}

	return &LocalPortMappingInfo{
		LocalPort:      localPort,
		ExternalPort:   localPort, // TODO: Allow mapping to a different port
		ExternalIP:     ip,
		RouterLocation: gateway.Location(),
	}, nil
}
