package nat

import (
	"github.com/jackpal/go-nat-pmp"
	"net"
	"strings"
	"time"
)

type pmp struct {
	client *natpmp.Client
}

func (p *pmp) ExternalIP() (net.IP, error) {
	response, err := p.client.GetExternalAddress()
	if err != nil {
		return nil, err
	}

	return response.ExternalIPAddress[:], nil
}

func (p *pmp) AddMapping(protocol string, externalPort, internalPort uint16, expiry time.Duration) error {
	_, err := p.client.AddPortMapping(strings.ToLower(protocol), int(internalPort), int(externalPort), int(expiry/time.Second))
	return err
}

func (p *pmp) DeleteMapping(protocol string, externalPort, internalPort uint16) (err error) {
	_, err = p.client.AddPortMapping(strings.ToLower(protocol), int(internalPort), 0, 0)
	return err
}

func NewPMP() Provider {
	gateways, err := activeGateways()
	if err != nil {
		panic(err)
	}

	found := make(chan *pmp, len(gateways))

	for i := range gateways {
		gateway := gateways[i]

		go func() {
			client := natpmp.NewClient(gateway)

			if _, err := client.GetExternalAddress(); err != nil {
				found <- nil
			} else {
				found <- &pmp{client: client}
			}
		}()
	}

	timeout := time.NewTimer(1 * time.Second)
	defer timeout.Stop()

	for range gateways {
		select {
		case client := <-found:
			if client != nil {
				return client
			}
		case <-timeout.C:
			return nil
		}
	}

	panic("natpmp: unable to find gateway that supports NAT-PMP")
}
