package nat

import (
	"github.com/huin/goupnp"
	"github.com/huin/goupnp/dcps/internetgateway1"
	"github.com/huin/goupnp/dcps/internetgateway2"
	"github.com/pkg/errors"
	"net"
	"strings"
	"time"
)

type upnp struct {
	host string

	client interface {
		GetExternalIPAddress() (string, error)
		AddPortMapping(string, uint16, string, uint16, string, bool, string, uint32) error
		DeletePortMapping(string, uint16, string) error
		GetNATRSIPStatus() (sip bool, nat bool, err error)
	}
}

func (u *upnp) ExternalIP() (net.IP, error) {
	raw, err := u.client.GetExternalIPAddress()
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(raw)
	if ip == nil {
		return nil, errors.Errorf("upnp: got invalid IP %+v", raw)
	}

	return ip, nil
}

func (u *upnp) AddMapping(protocol string, externalPort, internalPort uint16, lifetime time.Duration) error {
	ip, err := u.deviceInternalAddress()
	if err != nil {
		return nil
	}

	_ = u.DeleteMapping(protocol, externalPort, internalPort)

	return u.client.AddPortMapping("", externalPort, strings.ToUpper(protocol), internalPort, ip.String(), true, "noise", uint32(lifetime/time.Second))
}

func (u *upnp) DeleteMapping(protocol string, externalPort, internalPort uint16) error {
	return u.client.DeletePortMapping("", externalPort, strings.ToUpper(protocol))
}

func (u *upnp) deviceInternalAddress() (net.IP, error) {
	deviceAddress, err := net.ResolveUDPAddr("udp4", u.host)
	if err != nil {
		return nil, errors.Wrap(err, "upnp: failed to resolve device address")
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, errors.Wrap(err, "upnp: could not load network interfaces")
	}

	for _, iface := range interfaces {
		addresses, err := iface.Addrs()
		if err != nil {
			return nil, errors.Wrap(err, "upnp: could not list addresses of network interface")
		}

		for _, address := range addresses {
			if address, ok := address.(*net.IPNet); ok && address.Contains(deviceAddress.IP) {
				return address.IP, nil
			}
		}
	}

	return nil, errors.Errorf("upnp: could not find local address within device net range %v", deviceAddress)
}

func NewUPnP() Provider {
	found := make(chan *upnp, 2)

	// IGDv1
	go discover(found, internetgateway1.URN_WANConnectionDevice_1, func(device *goupnp.RootDevice, client goupnp.ServiceClient) *upnp {
		switch client.Service.ServiceType {
		case internetgateway1.URN_WANIPConnection_1:
			return &upnp{device.URLBase.Host, &internetgateway1.WANIPConnection1{ServiceClient: client}}
		case internetgateway1.URN_WANPPPConnection_1:
			return &upnp{device.URLBase.Host, &internetgateway1.WANPPPConnection1{ServiceClient: client}}
		}
		return nil
	})

	// IGDv2
	go discover(found, internetgateway2.URN_WANConnectionDevice_2, func(device *goupnp.RootDevice, client goupnp.ServiceClient) *upnp {
		switch client.Service.ServiceType {
		case internetgateway2.URN_WANIPConnection_1:
			return &upnp{device.URLBase.Host, &internetgateway2.WANIPConnection1{ServiceClient: client}}
		case internetgateway2.URN_WANIPConnection_2:
			return &upnp{device.URLBase.Host, &internetgateway2.WANIPConnection2{ServiceClient: client}}
		case internetgateway2.URN_WANPPPConnection_1:
			return &upnp{device.URLBase.Host, &internetgateway2.WANPPPConnection1{ServiceClient: client}}
		}
		return nil
	})

	for i := 0; i < cap(found); i++ {
		if c := <-found; c != nil {
			return c
		}
	}

	panic("upnp: unable to find gateway that supports UPnP")
}

func discover(out chan<- *upnp, target string, matcher func(*goupnp.RootDevice, goupnp.ServiceClient) *upnp) {
	devices, err := goupnp.DiscoverDevices(target)
	if err != nil {
		out <- nil
		return
	}

	found := false

	for i := 0; i < len(devices) && !found; i++ {
		if devices[i].Root == nil {
			continue
		}

		devices[i].Root.Device.VisitServices(func(service *goupnp.Service) {
			if found {
				return
			}

			client := goupnp.ServiceClient{
				SOAPClient: service.NewSOAPClient(),
				RootDevice: devices[i].Root,
				Location:   devices[i].Location,
				Service:    service,
			}

			client.SOAPClient.HTTPClient.Timeout = 3 * time.Second

			upnp := matcher(devices[i].Root, client)
			if upnp == nil {
				return
			}

			if _, nat, err := upnp.client.GetNATRSIPStatus(); err != nil || !nat {
				return
			}

			out <- upnp
			found = true
		})
	}

	if !found {
		out <- nil
	}
}
