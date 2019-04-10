package noise

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCloseMux(t *testing.T) {
	p := newPeer(nil, nil, nil, nil, nil)

	t.Run("does not allow for default mux to be closed", func(t *testing.T) {
		assert.Error(t, p.m.Close())
	})

	t.Run("de-registers mux from peer", func(t *testing.T) {
		m := p.Mux()
		assert.NoError(t, m.Close())
		assert.Nil(t, p.recv[m.id])
	})
}

func BenchmarkNewMux(b *testing.B) {
	p := newPeer(nil, nil, nil, nil, nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Mux()
	}
}
