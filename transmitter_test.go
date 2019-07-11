package lcm

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewTransmitter(t *testing.T) {
	t.Run("good_ip", func(t *testing.T) {
		addr, err := net.ResolveUDPAddr("udp", multicastIp)
		require.NoError(t, err)
		transmitter, err := NewTransmitter(addr)
		require.NoError(t, err)
		require.NoError(t, transmitter.Close())
	})
	t.Run("bad_ip", func(t *testing.T) {
		addr, err := net.ResolveUDPAddr("udp", nonMulticastIp)
		require.NoError(t, err)
		_, err = NewTransmitter(addr)
		require.Error(t, err)
	})
}

func TestSetWriteDeadline(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", multicastIp)
	require.NoError(t, err)
	transmitter, err := NewTransmitter(addr)
	require.NoError(t, err)
	t.Run("time_in_past", func(t *testing.T) {
		pastTime := time.Now()
		time.Sleep(sleepDuration)
		require.NoError(t, transmitter.SetWriteDeadline(pastTime))
	})
	t.Run("time_now", func(t *testing.T) {
		require.NoError(t, transmitter.SetWriteDeadline(time.Now()))
	})
	t.Run("time_in_future", func(t *testing.T) {
		require.NoError(t, transmitter.SetWriteDeadline(time.Now().Add(10*time.Hour)))
	})
	t.Run("closed_conn", func(t *testing.T) {
		require.NoError(t, transmitter.conn.Close())
		require.Error(t, transmitter.SetWriteDeadline(time.Now()))
	})
}

func TestTransmitterClose(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", multicastIp)
	require.NoError(t, err)
	transmitter, err := NewTransmitter(addr)
	require.NoError(t, err)
	t.Run("open_conn", func(t *testing.T) {
		require.NoError(t, transmitter.Close())
		require.Error(t, transmitter.conn.Close())
	})
	t.Run("closed_conn", func(t *testing.T) {
		transmitter, err = NewTransmitter(addr)
		require.NoError(t, err)
		require.NoError(t, transmitter.conn.Close())
		require.Error(t, transmitter.Close())
	})
}

func TestPublish(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", multicastIp)
	require.NoError(t, err)
	transmitter, err := NewTransmitter(addr)
	defer func() {
		require.NoError(t, transmitter.Close())
	}()
	msg := Message{
		Channel: "channel",
		Data:    []byte("data"),
	}
	t.Run("too_big_data", func(t *testing.T) {
		badMsg := msg
		badMsg.Data = make([]byte, shortMessageMaxSize)
		require.Error(t, transmitter.Publish(&badMsg))
	})
	t.Run("too_big_channel", func(t *testing.T) {
		bs := make([]byte, maxChannelNameLength+1)
		badMsg := msg
		badMsg.Channel = string(bs)
		require.Error(t, transmitter.Publish(&badMsg))
	})
	t.Run("sequence_number", func(t *testing.T) {
		require.Equal(t, uint32(0), transmitter.publishSequenceNumber)
		require.NoError(t, transmitter.Publish(&msg))
		require.Equal(t, uint32(1), transmitter.publishSequenceNumber)
	})
}
