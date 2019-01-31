package nat

import (
	"github.com/gortc/stun"
	"net"
)

var _ Provider = (*stunProvider)(nil)

type stunProvider struct{}

func NewSTUNProvider() *stunProvider {
	return new(stunProvider)
}

func (stunProvider) ExternalIP() (ip net.IP) {
	c, err := stun.Dial("udp", "stun.l.google.com:19302")
	if err != nil {
		panic(err)
	}

	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	if err := c.Do(message, func(res stun.Event) {
		if res.Error != nil {
			panic(res.Error)
		}

		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res.Message); err != nil {
			panic(err)
		}

		ip = xorAddr.IP
	}); err != nil {
		panic(err)
	}

	return
}
