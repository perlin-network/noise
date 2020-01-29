package noise_test

import (
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/stretchr/testify/assert"
	"io"
	"net"
	"strconv"
	"testing"
	"testing/quick"
)

func TestID_String(t *testing.T) {
	t.Parallel()

	f := func(publicKey noise.PublicKey, host net.IP, port uint16) bool {
		h := host.String() // Make-shift 'normalizeIP(net.IP)'.
		if h == "<nil>" {
			h = ""
		}

		id := noise.NewID(publicKey, host, port)

		if !assert.Equal(t,
			fmt.Sprintf(
				`{"public_key": "%s", "address": "%s"}`,
				publicKey, net.JoinHostPort(h, strconv.FormatUint(uint64(port), 10)),
			),
			id.String(),
		) {
			return false
		}

		return true
	}

	assert.NoError(t, quick.Check(f, nil))
}

func TestUnmarshalID(t *testing.T) {
	t.Parallel()

	_, err := noise.UnmarshalID(nil)
	assert.EqualError(t, err, io.ErrUnexpectedEOF.Error())

	_, err = noise.UnmarshalID(append(noise.ZeroPublicKey[:], 1))
	assert.EqualError(t, err, io.ErrUnexpectedEOF.Error())

	_, err = noise.UnmarshalID(append(noise.ZeroPublicKey[:], append(net.IPv6loopback, 1)...))
	assert.EqualError(t, err, io.ErrUnexpectedEOF.Error())

	_, err = noise.UnmarshalID(append(noise.ZeroPublicKey[:], append(net.IPv6loopback, 1, 2)...))
	assert.NoError(t, err)
}
