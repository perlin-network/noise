package noise

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

type dummmyProtocol struct {
	clientCall time.Time
	serverCall time.Time
}

func (d *dummmyProtocol) Client(info Info, ctx context.Context, auth string, conn net.Conn) (net.Conn, error) {
	d.clientCall = time.Now()
	return conn, nil
}

func (d *dummmyProtocol) Server(info Info, conn net.Conn) (net.Conn, error) {
	d.serverCall = time.Now()
	return conn, nil

}
func TestCredentials(t *testing.T) {
	p1 := &dummmyProtocol{}
	p2 := &dummmyProtocol{}
	c := NewCredentials("127.0.0.1", p1, p2)

	_, _, _ = c.ClientHandshake(context.Background(), "", nil)
	assert.NotEmpty(t, p1.clientCall)
	assert.NotEmpty(t, p2.clientCall)
	assert.True(t, p1.clientCall.Before(p2.clientCall))

	_, _, _ = c.ServerHandshake(nil)
	assert.NotEmpty(t, p1.serverCall)
	assert.NotEmpty(t, p2.serverCall)
	assert.True(t, p1.serverCall.Before(p2.serverCall))

	info := c.Info()
	assert.Equal(t, "noise", info.SecurityProtocol)
	assert.Equal(t, "0.0.1", info.SecurityVersion)
	assert.Equal(t, "127.0.0.1", info.ServerName)

	_ = c.OverrideServerName("127.0.0.2")
	assert.Equal(t, "127.0.0.2", c.Host)

	clone := c.Clone()
	assert.NotEqual(t, &c, &clone)
}
