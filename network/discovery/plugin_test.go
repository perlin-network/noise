package discovery

import (
	"testing"
	"time"

	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/peer"

	"github.com/stretchr/testify/assert"
)

var (
	p *Plugin
)

func init() {
	p = New(WithPuzzleEnabled(8, 8))
	kp, nonce := p.PerformPuzzle()
	p.id = peer.CreateID("tcp://localhost:8000", kp.PublicKey, peer.WithNonce(nonce))
}

func TestGetWeakSignature(t *testing.T) {
	t.Parallel()

	expiration := time.Now().Add(-1 * time.Second)

	signature, err := p.GetWeakSignature(expiration)
	assert.Equal(t, nil, err, "GetWeakSignature() expected to have no error, got: %v", err)
	assert.NotEqual(t, 0, len(signature), "GetWeakSignature() expected to have non-zero signature length")

	msg := &protobuf.Message{
		Expiration: expiration.UnixNano(),
	}

	// test no message signature
	ok, err := p.verifyWeakSignature(msg)
	assert.NotEqual(t, nil, err, "verifyWeakSignature() expected to have error")
	assert.Equal(t, false, ok, "verifyWeakSignature() expected to have incorrect signature")

	// test no message sender
	msg.Signature = signature
	ok, err = p.verifyWeakSignature(msg)
	assert.Equal(t, ErrSignatureNoSender, err, "verifyWeakSignature() expected to have error %v, got %v", ErrSignatureNoSender, err)
	assert.Equal(t, false, ok, "verifyWeakSignature() expected to have incorrect signature")

	// test expired signature
	msg.Sender = (*protobuf.ID)(&p.id)
	ok, err = p.verifyWeakSignature(msg)
	assert.Equal(t, ErrSignatureExpired, err, "verifyWeakSignature() expected %v, got %v", ErrSignatureExpired, err)
	assert.Equal(t, false, ok, "verifyWeakSignature() expected to have incorrect signature")

	// test invalid signature
	msg.Signature = []byte("invalid-signature")
	ok, err = p.verifyWeakSignature(msg)
	assert.Equal(t, ErrSignatureInvalid, err, "verifyWeakSignature() expected %v, got %v", ErrSignatureInvalid, err)
	assert.Equal(t, false, ok, "verifyWeakSignature() expected to have incorrect signature")

	expiration = time.Now().Add(5 * time.Second)

	signature, err = p.GetWeakSignature(expiration)
	assert.Equal(t, nil, err, "GetWeakSignature() expected to have no error, got: %v", err)
	assert.NotEqual(t, 0, len(signature), "GetWeakSignature() expected to have non-zero signature length")

	msg.Expiration = expiration.UnixNano()
	msg.Signature = signature

	ok, err = p.verifyWeakSignature(msg)
	assert.Equal(t, nil, err, "verifyWeakSignature() expected to not have error, got %+v", err)
	assert.Equal(t, true, ok, "verifyWeakSignature() expected to have correct signature")

}
