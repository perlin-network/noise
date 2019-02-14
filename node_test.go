package noise

import (
	"fmt"
	"github.com/perlin-network/noise/transport"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  parameters
		wantErr bool
	}{
		{
			name: "bad port 1",
			params: func() parameters {
				p := DefaultParams()
				p.Port = 1
				return p
			}(),
			wantErr: true,
		},
		{
			name: "bad port 1023",
			params: func() parameters {
				p := DefaultParams()
				p.Port = 1023
				return p
			}(),
			wantErr: true,
		},
		{
			name: "good port 0",
			params: func() parameters {
				p := DefaultParams()
				p.Port = 0
				return p
			}(),
			wantErr: false,
		},
		{
			name: "bad transport",
			params: func() parameters {
				p := DefaultParams()
				p.Transport = nil
				return p
			}(),
			wantErr: true,
		},
		{
			name: "many parameters",
			params: func() parameters {
				p := DefaultParams()
				p.Metadata["a"] = "b"
				p.Metadata["1"] = 2
				p.Metadata["f"] = 3.0
				return p
			}(),
			wantErr: false,
		},
		{
			name: "bad host",
			params: func() parameters {
				p := DefaultParams()
				p.Host = "bad host"
				p.Port = 1234
				return p
			}(),
			wantErr: true,
		},
		{
			name: "use mock transport",
			params: func() parameters {
				p := DefaultParams()
				p.Port = 1234
				p.Transport = transport.NewBuffered()
				return p
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewNode(tt.params)
			if tt.wantErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
			assert.NotNil(t, node)

			// check the metadata
			for key, val := range tt.params.Metadata {
				assert.Equal(t, val, node.Get(key))
			}

			// check the port
			if tt.params.Port == 0 {
				assert.True(t, node.Port() > 1024)
			} else {
				assert.Equal(t, tt.params.Port, node.Port())
			}
			assert.Equal(t, fmt.Sprintf("%s:%d", tt.params.Host, node.Port()), node.ExternalAddress())
		})
	}
}
