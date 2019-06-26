package lcm

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const multicastIp = "224.0.0.50:10000"
const nonMulticastIp = "10.0.0.1:0"
const sleepDuration = 100 * time.Millisecond

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

func TestPublishReceive(t *testing.T) {
	flowControlChan := make(chan struct{})
	defer close(flowControlChan)
	addr, err := net.ResolveUDPAddr("udp", multicastIp)
	require.NoError(t, err)
	listener, err := NewListener(addr)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, listener.Close())
	}()
	transmitter, err := NewTransmitter(addr)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, transmitter.Close())
	}()
	var receiveMsg Message
	t.Run("max_size_small_message", func(t *testing.T) {
		// Max size of data to fit in LCM message.
		bigMsgChannel := "channel"
		dataMaxSize := shortMessageMaxSize - shortHeaderSize - len([]byte(bigMsgChannel)) - 1
		bigMsgData := make([]byte, dataMaxSize)
		bigMsg := Message{
			Channel: bigMsgChannel,
			Data:    bigMsgData,
		}
		go func() {
			<-flowControlChan
			require.NoError(t, transmitter.Publish(&bigMsg))
		}()
		flowControlChan <- struct{}{}
		require.NoError(t, listener.Receive(&receiveMsg))
		require.Equal(t, bigMsgChannel, receiveMsg.Channel)
		require.Equal(t, bigMsgData, receiveMsg.Data)
	})
	t.Run("no_deadline", func(t *testing.T) {
		sendMsg := Message{
			Channel: "no deadline",
			Data:    []byte("no deadline"),
		}
		go func() {
			<-flowControlChan
			require.NoError(t, transmitter.Publish(&sendMsg))
		}()
		flowControlChan <- struct{}{}
		require.NoError(t, listener.Receive(&receiveMsg))
		require.Equal(t, sendMsg.Channel, receiveMsg.Channel)
		require.Equal(t, sendMsg.Data, receiveMsg.Data)
	})
	t.Run("read_deadline_in_future", func(t *testing.T) {
		sendMsg := Message{
			Channel: "future read deadline",
			Data:    []byte("future read deadline"),
		}
		go func() {
			<-flowControlChan
			require.NoError(t, transmitter.Publish(&sendMsg))
		}()
		require.NoError(t, listener.SetReadDeadline(time.Now().Add(1*time.Second)))
		flowControlChan <- struct{}{}
		require.NoError(t, listener.Receive(&receiveMsg))
		require.Equal(t, sendMsg.Channel, receiveMsg.Channel)
		require.Equal(t, sendMsg.Data, receiveMsg.Data)
	})
	t.Run("write_deadline_in_future", func(t *testing.T) {
		sendMsg := Message{
			Channel: "write deadline",
			Data:    []byte("write deadline"),
		}
		go func() {
			<-flowControlChan
			require.NoError(t, transmitter.SetWriteDeadline(time.Now().Add(1*time.Second)))
			require.NoError(t, transmitter.Publish(&sendMsg))
		}()
		flowControlChan <- struct{}{}
		require.NoError(t, listener.Receive(&receiveMsg))
		require.Equal(t, sendMsg.Channel, receiveMsg.Channel)
		require.Equal(t, sendMsg.Data, receiveMsg.Data)
	})
	t.Run("after_resetting_read_deadline", func(t *testing.T) {
		listener.SetReadDeadline(time.Now())
		time.Sleep(sleepDuration)
		listener.SetReadDeadline(time.Time{})
		sendMsg := Message{
			Channel: "reset read deadline",
			Data:    []byte("reset read deadline"),
		}
		go func() {
			<-flowControlChan
			require.NoError(t, transmitter.Publish(&sendMsg))
		}()
		flowControlChan <- struct{}{}
		require.NoError(t, listener.Receive(&receiveMsg))
		require.Equal(t, sendMsg.Channel, receiveMsg.Channel)
		require.Equal(t, sendMsg.Data, receiveMsg.Data)
	})
	t.Run("after_resetting_write_deadline", func(t *testing.T) {
		transmitter.SetWriteDeadline(time.Now())
		time.Sleep(sleepDuration)
		transmitter.SetWriteDeadline(time.Time{})
		sendMsg := Message{
			Channel: "reset write deadline",
			Data:    []byte("reset write deadline"),
		}
		go func() {
			<-flowControlChan
			require.NoError(t, transmitter.Publish(&sendMsg))
		}()
		flowControlChan <- struct{}{}
		require.NoError(t, listener.Receive(&receiveMsg))
		require.Equal(t, sendMsg.Channel, receiveMsg.Channel)
		require.Equal(t, sendMsg.Data, receiveMsg.Data)
	})
	t.Run("after_read_deadline", func(t *testing.T) {
		require.NoError(t, listener.SetReadDeadline(time.Now()))
		time.Sleep(sleepDuration)
		require.Error(t, listener.Receive(&receiveMsg))
	})
	t.Run("after_write_deadline", func(t *testing.T) {
		sendMsg := Message{
			Channel: "after write deadline",
			Data:    []byte("after write deadline"),
		}
		require.NoError(t, transmitter.SetWriteDeadline(time.Now()))
		time.Sleep(sleepDuration)
		require.Error(t, transmitter.Publish(&sendMsg))
	})
}

func TestUnmarshal(t *testing.T) {
	msg := Message{}
	t.Run("empty_data", func(t *testing.T) {
		err := msg.Unmarshal(make([]byte, 0))
		require.Equal(t, "to small to be an LCM message: 0", err.Error())
	})
	t.Run("bad_header_magic", func(t *testing.T) {
		data, n := createMessageData("channel", "payload")
		binary.BigEndian.PutUint32(data[indexOfShortHeaderMagic:], 0x00000000)
		err := msg.Unmarshal(data[:n])
		require.Equal(t, "invalid header magic: 0", err.Error())
	})
	t.Run("non_terminated_channel", func(t *testing.T) {
		data, n := createMessageData("channel", "payload")
		data[shortHeaderSize+len([]byte("channel"))] = 1
		err := msg.Unmarshal(data[:n])
		require.Equal(t, "invalid format for channel name, couldn't find string-termination", err.Error())
	})
}

func createMessageData(channel string, payload string) ([]byte, int) {
	data := make([]byte, shortHeaderSize+shortMessageMaxSize)
	c := []byte(channel)
	channelSize := len(channel)
	p := []byte(payload)
	binary.BigEndian.PutUint32(data[indexOfShortHeaderMagic:], shortHeaderMagic)
	binary.BigEndian.PutUint32(data[indexOfShortHeaderSequence:], uint32(0))
	copy(data[indexOfChannelName:], c)
	data[shortHeaderSize+channelSize] = 0
	copy(data[indexOfChannelName+channelSize+1:], p)
	size := shortHeaderSize + channelSize + 1 + len(p)
	return data, size
}
