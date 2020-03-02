package lcm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMessage_MarshalUnmarshal(t *testing.T) {
	for _, tt := range []struct {
		msg     string
		data    []byte
		message Message
	}{
		{
			msg: "no payload",
			data: []byte{
				0x4c, 0x43, 0x30, 0x32, // short header magic
				0x12, 0x34, 0x56, 0x78, // sequence number
				'a', 0x00, // channel
			},
			message: Message{
				SequenceNumber: 0x12345678,
				Channel:        "a",
				Data:           []byte{},
			},
		},
		{
			msg: "payload",
			data: []byte{
				0x4c, 0x43, 0x30, 0x32, // short header magic
				0x12, 0x34, 0x56, 0x78, // sequence number
				'a', 'b', 'c', 0x00, // channel
				0x01, 0x02, 0x03, // payload
			},
			message: Message{
				SequenceNumber: 0x12345678,
				Channel:        "abc",
				Data:           []byte{0x01, 0x02, 0x03},
			},
		},
		{
			msg: "payload channel with params",
			data: []byte{
				0x4c, 0x43, 0x30, 0x32, // short header magic
				0x12, 0x34, 0x56, 0x78, // sequence number
				'a', 'b', 'c', '?', 'z', '=', 'l', 'z', '4', 0x00, // channel
				0x01, 0x02, 0x03, // payload
			},
			message: Message{
				SequenceNumber: 0x12345678,
				Channel:        "abc",
				Params:         "z=lz4",
				Data:           []byte{0x01, 0x02, 0x03},
			},
		},
	} {
		tt := tt
		t.Run(tt.msg, func(t *testing.T) {
			t.Run("marshal", func(t *testing.T) {
				var data [lengthOfLargestUDPMessage]byte
				n, err := tt.message.marshal(data[:])
				require.NoError(t, err)
				require.Equal(t, len(tt.data), n)
				require.Equal(t, tt.data, data[:n])
			})
			t.Run("unmarshal", func(t *testing.T) {
				var msg Message
				require.NoError(t, msg.unmarshal(tt.data))
				require.Equal(t, tt.message, msg)
			})
		})
	}
}

func TestMessage_Unmarshal_Errors(t *testing.T) {
	for _, tt := range []struct {
		msg  string
		data []byte
		err  string
	}{
		{
			msg: "invalid size",
			data: []byte{
				0x4c, 0x43, 0x30, 0x32, // short header magic
				0x12, 0x34, 0x56,
			},
			err: "insufficient data: 7 bytes",
		},
		{
			msg: "invalid channel",
			data: []byte{
				0x4c, 0x43, 0x30, 0x32, // short header magic
				0x12, 0x34, 0x56, 0x78, // sequence number
				'a', 'b', 'c', 'd', // channel (missing null byte)
			},
			err: "invalid channel: not null-terminated",
		},
		{
			msg: "invalid magic",
			data: []byte{
				0xde, 0xad, 0xbe, 0xef, // short header magic
				0x12, 0x34, 0x56, 0x78, // sequence number
				'a', 'b', 'c', 'd', // channel (missing null byte)
			},
			err: "wrong header magic: 0xdeadbeef",
		},
	} {
		tt := tt
		t.Run(tt.msg, func(t *testing.T) {
			var msg Message
			err := msg.unmarshal(tt.data)
			require.NotNil(t, err)
			require.Equal(t, tt.err, err.Error())
		})
	}
}
