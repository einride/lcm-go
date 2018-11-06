package lcm

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProvider_Examples(t *testing.T) {
	for _, tt := range []struct {
		provider string
		expected Provider
	}{
		{
			provider: "udpm://239.255.76.67:7667",
			expected: &UDPMulticastProvider{Address: "239.255.76.67:7667"},
		},
		{
			provider: "udpm://239.255.76.67:7667?ttl=1",
			expected: &UDPMulticastProvider{Address: "239.255.76.67:7667", TTL: 1},
		},
		{
			provider: "udpm://239.255.76.67:7667?ttl=1&recv_buf_size=1024",
			expected: &UDPMulticastProvider{Address: "239.255.76.67:7667", TTL: 1, ReceiveBufferSize: 1024},
		},
	} {
		t.Run(fmt.Sprintf(tt.provider), func(t *testing.T) {
			actual, err := ParseProvider(tt.provider)
			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}
}
