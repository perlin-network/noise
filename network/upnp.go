package network

import (
	"github.com/NebulousLabs/go-upnp"
)

type LocalPortMappingInfo struct {
	LocalPort uint16
	ExternalPort uint16
	ExternalIP string
	RouterLocation string
}

func (m *LocalPortMappingInfo) Close() {
	d, err := upnp.Load(m.RouterLocation)
	if err == nil {
		d.Clear(m.ExternalPort)
	}
}

func AddPersistentLocalPortMapping(
	localPort uint16,
) (*LocalPortMappingInfo, error) {
	d, err := upnp.Discover()
	if err != nil {
		return nil, err
	}

	ip, err := d.ExternalIP()
	if err != nil {
		return nil, err
	}

	err = d.Forward(localPort, "Noise")
	if err != nil {
		return nil, err
	}

	return &LocalPortMappingInfo {
		LocalPort: localPort,
		ExternalPort: localPort, // TODO: Allow mapping to a different port
		ExternalIP: ip,
		RouterLocation: d.Location(),
	}, nil
}
