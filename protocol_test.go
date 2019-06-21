package noise

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/peer"
	"testing"
)

func TestInfo(t *testing.T) {
	p := &peer.Peer{}
	assert.Nil(t, InfoFromPeer(p))

	p.AuthInfo = Info{}

	info := InfoFromPeer(p)
	assert.NotNil(t, info)

	assert.Equal(t, "noise", info.AuthType())

	info.Put("key1", "val1")
	assert.Equal(t, "val1", info.Get("key1"))

	info.PutString("key2", "val2")
	assert.Equal(t, "val2", info.String("key2"))

	info.PutBytes("key3", []byte("val3"))
	assert.Equal(t, []byte("val3"), info.Bytes("key3"))
}
