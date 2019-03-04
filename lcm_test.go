package lcm_test

import (
	"encoding/binary"
	"github.com/einride/lcm-go"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestLCM_Publish(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "239.255.76.67:7667")
	require.NoError(t, err)
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	require.NoError(t, err)
	go func() {
		lc, err := lcm.Create("udpm://239.255.76.67:7667")
		require.NoError(t, err)
		require.NoError(t, lc.Publish("channel", []byte("payload")))
		require.NoError(t, lc.Close())
	}()
	data := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(data)
	require.NoError(t, err)
	expected := []byte{
		0x4c, 0x43, 0x30, 0x32, // magic
		0x00, 0x00, 0x00, 0x00, // sequence number
	}
	expected = append(expected, []byte("channel")...) // channel
	expected = append(expected, 0)                    // null string terminator
	expected = append(expected, []byte("payload")...) // payload
	require.Equal(t, expected, data[:n])
}

func TestLCM_Publish_Multiple(t *testing.T) {
	n := 100
	addr, err := net.ResolveUDPAddr("udp", "239.255.76.67:7667")
	require.NoError(t, err)
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	require.NoError(t, err)
	go func() {
		lc, err := lcm.Create("udpm://239.255.76.67:7667")
		require.NoError(t, err)
		for i := 0; i < n; i++ {
			require.NoError(t, lc.Publish("channel", []byte("payload")))
		}
		require.NoError(t, lc.Close())
	}()
	data := make([]byte, 1024)
	for i := 0; i < n; i++ {
		_, _, err := conn.ReadFromUDP(data)
		require.NoError(t, err)
		require.Equal(t, uint32(i), binary.BigEndian.Uint32(data[4:]))
	}
}

func TestLCM_ReadMessage(t *testing.T) {
	go func() {
		lc, err := lcm.Create("udpm://239.255.76.67:7667")
		require.NoError(t, err)
		require.NoError(t, lc.Publish("channel", []byte("payload")))
		require.NoError(t, lc.Close())
	}()
	addr, err := net.ResolveUDPAddr("udp", "239.255.76.67:7667")
	require.NoError(t, err)
	reader, err := net.ListenMulticastUDP("udp", nil, addr)
	require.NoError(t, err)
	m, err := lcm.ReadMessage(reader)
	require.NoError(t, err)
	require.Equal(t, uint32(0), m.Sequence)
	require.Equal(t, "channel", m.Topic)
	expected := []byte("payload")
	require.Equal(t, expected, m.Data)
}

func TestLCM_ReadMessage_Multiple(t *testing.T) {
	n := 100
	go func() {
		lc, err := lcm.Create("udpm://239.255.76.67:7667")
		require.NoError(t, err)
		for i := 0; i < n; i++ {
			require.NoError(t, lc.Publish("channel", []byte("payload")))
		}
		require.NoError(t, lc.Close())
	}()
	addr, err := net.ResolveUDPAddr("udp", "239.255.76.67:7667")
	require.NoError(t, err)
	reader, err := net.ListenMulticastUDP("udp", nil, addr)
	require.NoError(t, err)
	for i := 0; i < n; i++ {
		m, err := lcm.ReadMessage(reader)
		require.NoError(t, err)
		require.Equal(t, uint32(i), m.Sequence)
		expected := []byte("payload")
		require.Equal(t, expected, m.Data)
	}
}
