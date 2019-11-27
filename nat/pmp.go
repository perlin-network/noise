// Copyright (c) 2019 Perlin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
	_, err := p.client.AddPortMapping(
		strings.ToLower(protocol), int(internalPort), int(externalPort), int(expiry/time.Second),
	)

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
