package lcm

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

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
