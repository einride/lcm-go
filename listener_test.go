package lcm

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewListener(t *testing.T) {
	t.Run("good_ip", func(t *testing.T) {
		addr, err := net.ResolveUDPAddr("udp", multicastIp)
		require.NoError(t, err)
		l, err := NewListener(addr)
		require.NoError(t, err)
		require.NoError(t, l.Close())
	})
	t.Run("bad_ip", func(t *testing.T) {
		addr, err := net.ResolveUDPAddr("udp", nonMulticastIp)
		require.NoError(t, err)
		_, err = NewListener(addr)
		require.Error(t, err)
	})
}

func TestSetReadDeadline(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", multicastIp)
	require.NoError(t, err)
	listener, err := NewListener(addr)
	require.NoError(t, err)
	t.Run("time_in_past", func(t *testing.T) {
		pastTime := time.Now()
		time.Sleep(sleepDuration)
		require.NoError(t, listener.SetReadDeadline(pastTime))
	})
	t.Run("time_now", func(t *testing.T) {
		require.NoError(t, listener.SetReadDeadline(time.Now()))
	})
	t.Run("time_in_future", func(t *testing.T) {
		require.NoError(t, listener.SetReadDeadline(time.Now().Add(10*time.Hour)))
	})
	t.Run("closed_conn", func(t *testing.T) {
		require.NoError(t, listener.conn.Close())
		require.Error(t, listener.SetReadDeadline(time.Now()))
	})
}

func TestListenerClose(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", multicastIp)
	require.NoError(t, err)
	listener, err := NewListener(addr)
	require.NoError(t, err)
	t.Run("open_conn", func(t *testing.T) {
		require.NoError(t, listener.Close())
		require.Error(t, listener.conn.Close())
	})
	t.Run("closed_conn", func(t *testing.T) {
		listener, err = NewListener(addr)
		require.NoError(t, err)
		require.NoError(t, listener.conn.Close())
		require.Error(t, listener.Close())
	})
}

func TestReceive_BadMessage(t *testing.T) {
	flowControlChan := make(chan struct{})
	addr, err := net.ResolveUDPAddr("udp", multicastIp)
	require.NoError(t, err)
	conn, err := net.DialUDP("udp", nil, addr)
	require.NoError(t, err)
	listener, err := NewListener(addr)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, listener.Close())
	}()
	go func() {
		<-flowControlChan
		_, err = conn.Write(make([]byte, 1000))
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	}()
	close(flowControlChan)
	require.Error(t, listener.Receive(&Message{}))
}
