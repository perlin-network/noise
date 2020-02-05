package noise_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func TestSendUnderMaxNumConnections(t *testing.T) {
	defer goleak.VerifyNone(t)

	var nodes []*noise.Node

	count := 10
	expected := count * (count - 1)

	var wg sync.WaitGroup
	wg.Add(expected)

	for i := 0; i < count; i++ {
		node, err := noise.NewNode(noise.WithNodeMaxOutboundConnections(1))
		assert.NoError(t, err)

		defer node.Close()

		node.Handle(func(ctx noise.HandlerContext) error {
			wg.Done()
			return nil
		})

		assert.NoError(t, node.Listen())

		nodes = append(nodes, node)
	}

	for i, x := range nodes {
		for j, y := range nodes {
			if i == j {
				continue
			}

			assert.NoError(t, x.Send(context.TODO(), y.Addr(), []byte(fmt.Sprintf("hello %d! i'm %d!", j, i))))
		}
	}

	wg.Wait()
}

func TestSendWithTwoPeersUnderMaxNumConnections(t *testing.T) {
	defer goleak.VerifyNone(t)

	count := 1000

	var wg sync.WaitGroup
	wg.Add(count)

	a, err := noise.NewNode(noise.WithNodeMaxOutboundConnections(1))
	assert.NoError(t, err)

	defer a.Close()

	a.Handle(func(ctx noise.HandlerContext) error {
		wg.Done()
		return nil
	})

	b, err := noise.NewNode()
	assert.NoError(t, err)

	defer b.Close()

	b.Handle(func(ctx noise.HandlerContext) error {
		wg.Done()
		return nil
	})

	assert.NoError(t, a.Listen())
	assert.NoError(t, b.Listen())

	for i := 0; i < count; i++ {
		if i%2 == 0 {
			assert.NoError(t, a.Send(context.TODO(), b.Addr(), []byte("hello b!")))
		} else {
			assert.NoError(t, b.Send(context.TODO(), a.Addr(), []byte("hello a!")))
		}
	}

	wg.Wait()
}

func TestRPCUnderMaxNumConnections(t *testing.T) {
	defer goleak.VerifyNone(t)

	var nodes []*noise.Node

	count := 10

	for i := 0; i < count; i++ {
		node, err := noise.NewNode(noise.WithNodeMaxOutboundConnections(1))
		assert.NoError(t, err)

		defer node.Close()

		node.Handle(func(ctx noise.HandlerContext) error {
			return ctx.Send(ctx.Data())
		})

		assert.NoError(t, node.Listen())

		nodes = append(nodes, node)
	}

	var wg sync.WaitGroup
	wg.Add(count * (count - 1))

	for i, x := range nodes {
		for j, y := range nodes {
			i, j, x, y := i, j, x, y

			if i == j {
				continue
			}

			go func() {
				data, err := x.Request(context.TODO(), y.Addr(), []byte("hello!"))
				assert.EqualValues(t, data, []byte("hello!"))
				assert.NoError(t, err)

				wg.Done()
			}()
		}
	}

	wg.Wait()
}

func TestRPCWithTwoPeersUnderMaxNumConnections(t *testing.T) {
	defer goleak.VerifyNone(t)

	count := 100

	a, err := noise.NewNode(noise.WithNodeMaxOutboundConnections(1))
	assert.NoError(t, err)

	defer a.Close()

	a.Handle(func(ctx noise.HandlerContext) error {
		assert.EqualValues(t, ctx.Data(), []byte("hello a!"))
		return ctx.Send([]byte("hello b!"))
	})

	b, err := noise.NewNode()
	assert.NoError(t, err)

	defer b.Close()

	b.Handle(func(ctx noise.HandlerContext) error {
		assert.EqualValues(t, ctx.Data(), []byte("hello b!"))
		return ctx.Send([]byte("hello a!"))
	})

	assert.NoError(t, a.Listen())
	assert.NoError(t, b.Listen())

	for i := 0; i < count; i++ {
		if i%2 == 0 {
			data, err := a.Request(context.TODO(), b.Addr(), []byte("hello b!"))
			assert.EqualValues(t, data, []byte("hello a!"))
			assert.NoError(t, err)
		} else {
			data, err := b.Request(context.TODO(), a.Addr(), []byte("hello a!"))
			assert.EqualValues(t, data, []byte("hello b!"))
			assert.NoError(t, err)
		}
	}
}

func TestCloseClientFromServerSide(t *testing.T) {
	defer goleak.VerifyNone(t)

	a, err := noise.NewNode()
	assert.NoError(t, err)

	defer a.Close()

	b, err := noise.NewNode()
	assert.NoError(t, err)

	defer b.Close()

	assert.NoError(t, a.Listen())
	assert.NoError(t, b.Listen())

	assert.NoError(t, b.Send(context.Background(), a.Addr(), []byte("hello")))

	assert.Len(t, a.Inbound(), 1)
	assert.Len(t, b.Outbound(), 1)

	ab, ba := a.Inbound()[0], b.Outbound()[0]

	ab.Close()
	ab.WaitUntilClosed()
	ba.WaitUntilClosed()

	assert.Len(t, a.Inbound(), 0)
	assert.Len(t, b.Outbound(), 0)
}

func TestCloseClientFromClientSide(t *testing.T) {
	defer goleak.VerifyNone(t)

	a, err := noise.NewNode()
	assert.NoError(t, err)

	defer a.Close()

	b, err := noise.NewNode()
	assert.NoError(t, err)

	defer b.Close()

	assert.NoError(t, a.Listen())
	assert.NoError(t, b.Listen())

	_, err = b.Ping(context.Background(), a.Addr())
	assert.NoError(t, err)

	assert.Len(t, a.Inbound(), 1)
	assert.Len(t, b.Outbound(), 1)

	ab, ba := a.Inbound()[0], b.Outbound()[0]

	ba.Close()
	ba.WaitUntilClosed()
	ab.WaitUntilClosed()

	assert.Len(t, a.Inbound(), 0)
	assert.Len(t, b.Outbound(), 0)
}

func TestIdleTimeoutServerSide(t *testing.T) {
	defer goleak.VerifyNone(t)

	a, err := noise.NewNode(noise.WithNodeIdleTimeout(50 * time.Millisecond))
	assert.NoError(t, err)

	defer a.Close()

	b, err := noise.NewNode()
	assert.NoError(t, err)

	defer b.Close()

	assert.NoError(t, a.Listen())
	assert.NoError(t, b.Listen())

	_, err = b.Ping(context.Background(), a.Addr())
	assert.NoError(t, err)

	assert.Len(t, a.Inbound(), 1)
	assert.Len(t, b.Outbound(), 1)

	ab, ba := a.Inbound()[0], b.Outbound()[0]

	ba.WaitUntilClosed()
	ab.WaitUntilClosed()

	var abError *net.OpError
	assert.True(t, errors.As(ab.Error(), &abError) && abError.Timeout() || ab.Error() == io.EOF)

	assert.EqualValues(t, ba.Error(), io.EOF)

	assert.Len(t, a.Inbound(), 0)
	assert.Len(t, b.Outbound(), 0)
}

func TestIdleTimeoutClientSide(t *testing.T) {
	defer goleak.VerifyNone(t)

	a, err := noise.NewNode()
	assert.NoError(t, err)

	defer a.Close()

	b, err := noise.NewNode(noise.WithNodeIdleTimeout(50 * time.Millisecond))
	assert.NoError(t, err)

	defer b.Close()

	assert.NoError(t, a.Listen())
	assert.NoError(t, b.Listen())

	_, err = b.Ping(context.Background(), a.Addr())
	assert.NoError(t, err)

	assert.Len(t, a.Inbound(), 1)
	assert.Len(t, b.Outbound(), 1)

	ab, ba := a.Inbound()[0], b.Outbound()[0]

	ba.WaitUntilClosed()
	ab.WaitUntilClosed()

	var baError *net.OpError
	assert.True(t, errors.As(ba.Error(), &baError) && baError.Timeout() || ba.Error() == io.EOF)

	assert.EqualValues(t, ab.Error(), io.EOF)

	assert.Len(t, a.Inbound(), 0)
	assert.Len(t, b.Outbound(), 0)
}

func TestHandlerErrorCausesConnToClose(t *testing.T) {
	defer goleak.VerifyNone(t)

	a, err := noise.NewNode()
	assert.NoError(t, err)

	defer a.Close()

	expected := errors.New("ack")

	a.Handle(func(ctx noise.HandlerContext) error {
		return expected
	})

	b, err := noise.NewNode()
	assert.NoError(t, err)

	defer b.Close()

	assert.NoError(t, a.Listen())
	assert.NoError(t, b.Listen())

	assert.NoError(t, b.Send(context.TODO(), a.Addr(), []byte("hello")))

	ab, ba := a.Inbound()[0], b.Outbound()[0]

	ab.WaitUntilClosed()
	ba.WaitUntilClosed()

	assert.Equal(t, expected, ab.Error())
	assert.Equal(t, io.EOF, ba.Error())

	assert.Len(t, a.Inbound(), 0)
	assert.Len(t, b.Outbound(), 0)
}

func TestWithNodeMaxRecvMessageSize(t *testing.T) {
	// Set the limit to 1MB.

	a, err := noise.NewNode(noise.WithNodeMaxRecvMessageSize(1 << 20))
	assert.NoError(t, err)
	defer a.Close()

	assert.NoError(t, a.Listen())

	// Send a message that is just 1 byte over 1MB (minus 35 because of overhead from encrypting data).

	if err = a.Send(context.Background(), a.Addr(), make([]byte, (1<<20)-35)); err != nil {
		return
	}

	if inbound := a.Inbound(); len(inbound) > 0 {
		for _, client := range inbound {
			client.WaitUntilClosed()
		}
	}
}

func BenchmarkRPC(b *testing.B) {
	a, err := noise.NewNode()
	assert.NoError(b, err)

	defer a.Close()

	a.Handle(func(ctx noise.HandlerContext) error {
		return ctx.Send(ctx.Data())
	})

	assert.NoError(b, a.Listen())

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = a.Request(context.TODO(), a.Addr(), []byte("hello"))
		}
	})
}

func BenchmarkSend(b *testing.B) {
	a, err := noise.NewNode()
	assert.NoError(b, err)

	defer a.Close()

	a.Handle(func(ctx noise.HandlerContext) error {
		return nil
	})

	assert.NoError(b, a.Listen())

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = a.Send(context.TODO(), a.Addr(), []byte("hello"))
		}
	})
}
