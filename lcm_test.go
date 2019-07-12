package lcm

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestTransmitReceive(t *testing.T) {
	if testing.Short() {
		t.Skip("to avoid network connections and sleep overhead")
	}
	addr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 50), Port: 10000}
	var rx *Receiver
	t.Run("bind receiver", func(t *testing.T) {
		rxConn, err := net.ListenMulticastUDP("udp", nil, addr)
		require.NoError(t, err)
		rx = NewReceiver(rxConn)
	})
	defer func() {
		require.NoError(t, rx.Close())
	}()
	var tx *Transmitter
	t.Run("connect transmitter", func(t *testing.T) {
		txConn, err := net.DialUDP("udp", nil, addr)
		require.NoError(t, err)
		tx = NewTransmitter(txConn)
	})
	defer func() {
		require.NoError(t, tx.Close())
	}()
	t.Run("max size small message", func(t *testing.T) {
		// Max size of data to fit in LCM message.
		channel := t.Name()
		dataMaxSize := lengthOfLargestPayload - len([]byte(channel)) - 1
		data := make([]byte, dataMaxSize)
		var g errgroup.Group
		g.Go(func() error {
			return rx.Receive(context.Background())
		})
		require.NoError(t, tx.Transmit(context.Background(), channel, data))
		require.NoError(t, g.Wait())
		require.Equal(t, channel, rx.Message().Channel)
		require.Equal(t, data, rx.Message().Data)
	})
	t.Run("read deadline in future", func(t *testing.T) {
		channel := t.Name()
		data := []byte(t.Name())
		var g errgroup.Group
		g.Go(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			return rx.Receive(ctx)
		})
		require.NoError(t, tx.Transmit(context.Background(), channel, data))
		require.NoError(t, g.Wait())
		require.Equal(t, channel, rx.Message().Channel)
		require.Equal(t, data, rx.Message().Data)
	})
	t.Run("write deadline in future", func(t *testing.T) {
		channel := t.Name()
		data := []byte(t.Name())
		var g errgroup.Group
		g.Go(func() error {
			return rx.Receive(context.Background())
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, tx.Transmit(ctx, channel, data))
		require.NoError(t, g.Wait())
		require.Equal(t, channel, rx.Message().Channel)
		require.Equal(t, data, rx.Message().Data)
	})
	t.Run("read deadline exceeded", func(t *testing.T) {
		const duration = 100 * time.Millisecond
		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()
		time.Sleep(duration + time.Millisecond)
		require.Error(t, rx.Receive(ctx))
	})
	t.Run("write deadline exceeded", func(t *testing.T) {
		const duration = 100 * time.Millisecond
		channel := t.Name()
		data := []byte(t.Name())
		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()
		time.Sleep(duration + time.Millisecond)
		require.Error(t, tx.Transmit(ctx, channel, data))
	})
}
