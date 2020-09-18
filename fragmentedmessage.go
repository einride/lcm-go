package lcm

import (
	"encoding/binary"
	"fmt"
)

// https://lcm-proj.github.io/udp_multicast_protocol.html
// 0      7 8     15 16    23 24    31
// +--------+--------+--------+--------+
// | fragment_header_magic             |
// +--------+--------+--------+--------+
// | sequence_number                   |
// +--------+--------+--------+--------+
// | payload_size                      |
// +--------+--------+--------+--------+
// | fragment_offset                   |
// +--------+--------+--------+--------+
// | fragment_number | n_fragments     |
// +--------+--------+--------+--------+

// Message represents an LCM message.
type FragmentMessage struct {
	Message
	FragmentOffset uint32
	FragmentIndex  uint16
	TotalFragments uint16
}

// message structure constants.
const (
	indexOfFragmentHeaderMagic     = 0
	lengthOfFragmentHeaderMagic    = 4
	indexOfFragmentSequenceNumber  = indexOfFragmentHeaderMagic + lengthOfFragmentHeaderMagic
	lengthOfFragmentSequenceNumber = 4
	indexOfFragmentPayloadSize     = indexOfFragmentSequenceNumber + lengthOfFragmentSequenceNumber
	lengthOfFragmentPayloadSize    = 4
	indexOfFragmentOffset          = indexOfFragmentPayloadSize + lengthOfFragmentPayloadSize
	lengthOfFragmentOffset         = 4
	indexOfFragmentNumber          = indexOfFragmentOffset + lengthOfFragmentOffset
	lengthOfFragmentNumber         = 2
	indexOfFragmentCount           = indexOfFragmentNumber + lengthOfFragmentNumber
	lengthOfFragmentCount          = 2
	indexOfFragmentChannel         = indexOfFragmentCount + lengthOfFragmentCount
)

const (
	fragmentMessageMagic uint32 = 0x4c433033
	ethernetMTU                 = 1500
)

func (m *FragmentMessage) marshal(b []byte) (int, error) {
	cLen := len(m.Message.Channel)
	if cLen > lengthOfLongestChannel {
		return 0, fmt.Errorf("channel too long: %v bytes", len(m.Channel))
	}
	payloadSize := cLen + 1 + len(m.Data)
	if payloadSize > ethernetMTU {
		return 0, fmt.Errorf("channel and data too long: %v bytes", payloadSize)
	}
	binary.BigEndian.PutUint32(b[indexOfFragmentHeaderMagic:], fragmentMessageMagic)
	binary.BigEndian.PutUint32(b[indexOfFragmentSequenceNumber:], m.SequenceNumber)
	binary.BigEndian.PutUint32(b[indexOfFragmentPayloadSize:], uint32(len(m.Data)))
	binary.BigEndian.PutUint32(b[indexOfFragmentOffset:], m.FragmentOffset)
	binary.BigEndian.PutUint16(b[indexOfFragmentNumber:], m.FragmentIndex)
	binary.BigEndian.PutUint16(b[indexOfFragmentCount:], m.TotalFragments)
	if m.FragmentIndex == 0 {
		copy(b[indexOfFragmentChannel:], m.Channel)
		b[indexOfFragmentChannel+cLen] = 0
		copy(b[indexOfFragmentChannel+cLen+1:], m.Data)
		return indexOfFragmentChannel + cLen + payloadSize, nil
	}
	copy(b[indexOfFragmentCount+cLen+1:], m.Data)
	return indexOfFragmentCount + payloadSize, nil
}
