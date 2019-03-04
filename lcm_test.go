package lcm_test

import (
	"net"
	"testing"
	"time"

	"github.com/einride/lcm-go"
	"github.com/stretchr/testify/require"
)

func TestLCM_PublishReceive(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "239.255.76.67:7667")
	require.NoError(t, err)
	sendMsg := lcm.Message{
		Channel: "channel",
		Data:    []byte("payload"),
	}
	go func() {
		transmitter, err := lcm.NewTransmitter(addr)
		defer func() {
			require.NoError(t, transmitter.Close())
		}()
		require.NoError(t, err)
		require.NoError(t, transmitter.Publish(&sendMsg))
	}()
	listener, err := lcm.NewListener(addr)
	defer func() {
		require.NoError(t, listener.Close())
	}()
	require.NoError(t, err)
	var receiveMsg lcm.Message
	require.NoError(t, listener.Receive(&receiveMsg))
	require.Equal(t, sendMsg.Channel, receiveMsg.Channel)
	require.Equal(t, uint32(0), receiveMsg.SequenceNumber)
	require.Equal(t, sendMsg.Data, receiveMsg.Data)
}

func TestLCM_PublishReceive_Multiple(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "239.255.76.67:7667")
	require.NoError(t, err)
	sendMsg := lcm.Message{
		Channel: "channel",
		Data:    []byte("payload"),
	}
	n := 100
	go func() {
		transmitter, err := lcm.NewTransmitter(addr)
		defer func() {
			require.NoError(t, transmitter.Close())
		}()
		require.NoError(t, err)
		for i := 0; i < n; i++ {
			require.NoError(t, transmitter.Publish(&sendMsg))
		}
	}()
	listener, err := lcm.NewListener(addr)
	defer func() {
		require.NoError(t, listener.Close())
	}()
	require.NoError(t, err)
	var receiveMsg lcm.Message
	for i := 0; i < n; i++ {
		require.NoError(t, listener.Receive(&receiveMsg))
		require.Equal(t, sendMsg.Channel, receiveMsg.Channel)
		require.Equal(t, uint32(i), receiveMsg.SequenceNumber)
		require.Equal(t, sendMsg.Data, receiveMsg.Data)
	}
}

func TestLCM_ReceiveWithTimeout(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "239.255.76.67:7667")
	require.NoError(t, err)
	sendMsg := lcm.Message{
		Channel: "channel",
		Data:    []byte("payload"),
	}
	go func() {
		transmitter, err := lcm.NewTransmitter(addr)
		defer func() {
			require.NoError(t, transmitter.Close())
		}()
		require.NoError(t, err)
		require.NoError(t, transmitter.Publish(&sendMsg))
		require.NoError(t, transmitter.Publish(&sendMsg))
	}()
	listener, err := lcm.NewListener(addr)
	defer func() {
		require.NoError(t, listener.Close())
	}()
	require.NoError(t, err)
	var receiveMsg lcm.Message
	require.NoError(t, listener.SetReadDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, listener.Receive(&receiveMsg))
	require.Equal(t, sendMsg.Channel, receiveMsg.Channel)
	require.Equal(t, uint32(0), receiveMsg.SequenceNumber)
	require.Equal(t, sendMsg.Data, receiveMsg.Data)
	// Clear read-deadline
	require.NoError(t, listener.SetReadDeadline(time.Time{}))
	require.NoError(t, listener.Receive(&receiveMsg))
	require.Equal(t, sendMsg.Channel, receiveMsg.Channel)
	require.Equal(t, uint32(1), receiveMsg.SequenceNumber)
	require.Equal(t, sendMsg.Data, receiveMsg.Data)
}

func TestLCM_PublishReceive_MaxSizeSmallMessage(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "239.255.76.67:7667")
	require.NoError(t, err)
	// Max size of data to fit in a UDP packet (65507 - header(8) - length of channel - 1 (channel is 0-terminated)
	dataSize := 65507 - 8 - len([]byte("channel")) - 1
	data := make([]byte, dataSize)
	sendMsg := lcm.Message{
		Channel: "channel",
		Data:    data,
	}
	go func() {
		transmitter, err := lcm.NewTransmitter(addr)
		defer func() {
			require.NoError(t, transmitter.Close())
		}()
		require.NoError(t, err)
		require.NoError(t, transmitter.Publish(&sendMsg))
	}()
	listener, err := lcm.NewListener(addr)
	defer func() {
		require.NoError(t, listener.Close())
	}()
	require.NoError(t, err)
	var receiveMsg lcm.Message
	require.NoError(t, listener.Receive(&receiveMsg))
	require.Equal(t, sendMsg.Channel, receiveMsg.Channel)
	require.Equal(t, uint32(0), receiveMsg.SequenceNumber)
	require.Equal(t, sendMsg.Data, receiveMsg.Data)
}

func TestLCM_Publish_EmptyPayload(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "239.255.76.67:7667")
	require.NoError(t, err)
	sendMsg := lcm.Message{
		Channel: "channel",
		Data:    make([]byte, 0),
	}
	go func() {
		transmitter, err := lcm.NewTransmitter(addr)
		defer func() {
			require.NoError(t, transmitter.Close())
		}()
		require.NoError(t, err)
		require.NoError(t, transmitter.Publish(&sendMsg))
	}()
	listener, err := lcm.NewListener(addr)
	defer func() {
		require.NoError(t, listener.Close())
	}()
	require.NoError(t, err)
	var receiveMsg lcm.Message
	require.NoError(t, listener.Receive(&receiveMsg))
	require.Equal(t, sendMsg.Channel, receiveMsg.Channel)
	require.Equal(t, uint32(0), receiveMsg.SequenceNumber)
	require.Equal(t, sendMsg.Data, receiveMsg.Data)
}

func TestLCM_Publish_BadMessage(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "239.255.76.67:7667")
	require.NoError(t, err)
	transmitter, err := lcm.NewTransmitter(addr)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, transmitter.Close())
	}()
	t.Run("test_bad_payload", func(t *testing.T) {
		data := make([]byte, 65500)
		sendMsg := lcm.Message{
			Channel: "channel",
			Data:    data,
		}
		require.Error(t, transmitter.Publish(&sendMsg))
	})
	t.Run("test_bad_channel", func(t *testing.T) {
		sendMsg := lcm.Message{
			Channel: "a really really really really really really really long channel name (I won't fit!)",
			Data:    []byte("payload"),
		}
		require.Error(t, transmitter.Publish(&sendMsg))
	})
}

func TestLCM_Receive_BadMessage(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "239.255.76.68:7667")
	require.NoError(t, err)
	go func() {
		conn, err := net.DialUDP("udp", nil, addr)
		require.NoError(t, err)
		defer func() {
			require.NoError(t, conn.Close())
		}()
		_, err = conn.Write([]byte("not a valid lcm message"))
		require.NoError(t, err)
	}()
	listener, err := lcm.NewListener(addr)
	defer func() {
		require.NoError(t, listener.Close())
	}()
	require.NoError(t, err)
	require.Error(t, listener.Receive(&lcm.Message{}))
}

func TestLCM_NewConn_NotMulticast(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "192.168.1.123:7667")
	require.NoError(t, err)
	t.Run("test_transmitter_not_multicast", func(t *testing.T) {
		// Not a multicast address
		_, err := lcm.NewTransmitter(addr)
		require.Error(t, err)
	})
	t.Run("test_listener_bad_ip", func(t *testing.T) {
		// Not a multicast address
		_, err := lcm.NewListener(addr)
		require.Error(t, err)
	})
}
