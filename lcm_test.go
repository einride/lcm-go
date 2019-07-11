package lcm

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const multicastIp = "224.0.0.50:10000"
const nonMulticastIp = "10.0.0.1:0"
const sleepDuration = 100 * time.Millisecond

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
