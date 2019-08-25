package lcm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/bpf"
)

func TestShortMessageChannelFilter(t *testing.T) {
	for _, tt := range []struct {
		name     string
		program  []bpf.Instruction
		packet   []byte
		expected int
	}{
		{
			name:    "accepted 1",
			program: ShortMessageChannelFilter("foo", "barbaz"),
			packet: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // UDP header
				0x4c, 0x43, 0x30, 0x32, // magic
				0x00, 0x00, 0x00, 0x01, // sequence number
				'f', 'o', 'o', 0, // channel
			},
			expected: 0xffff,
		},
		{
			name:    "accepted 2",
			program: ShortMessageChannelFilter("foo", "barbaz"),
			packet: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // UDP header
				0x4c, 0x43, 0x30, 0x32, // magic
				0x00, 0x00, 0x00, 0x01, // sequence number
				'b', 'a', 'r', 'b', 'a', 'z', 0, // channel
			},
			expected: 0xffff,
		},
		{
			name:    "rejected due to wrong channel",
			program: ShortMessageChannelFilter("foo", "barbaz"),
			packet: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // UDP header
				0x4c, 0x43, 0x30, 0x32, // magic
				0x00, 0x00, 0x00, 0x01, // sequence number
				'b', 'a', 'r', 0, // channel
			},
			expected: 0,
		},
		{
			name:    "rejected due to wrong header magic",
			program: ShortMessageChannelFilter("foo", "barbaz"),
			packet: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // UDP header
				0x4c, 0x43, 0x30, 0x00, // wrong magic
				0x00, 0x00, 0x00, 0x01, // sequence number
				'f', 'o', 'o', 0, // channel
			},
			expected: 0,
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			vm, err := bpf.NewVM(tt.program)
			require.NoError(t, err)
			n, err := vm.Run(tt.packet)
			require.NoError(t, err)
			require.Equal(t, tt.expected, n)
		})
	}
}
