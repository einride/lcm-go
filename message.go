package lcm

import (
	"bytes"
	"encoding/binary"

	"golang.org/x/xerrors"
)

// lengthOfLargestUDPMessage is length in bytes of the largest possible UDP message.
const lengthOfLargestUDPMessage = 0xffff

// message structure constants.
const (
	indexOfHeaderMagic     = 0
	lengthOfHeaderMagic    = 4
	indexOfSequenceNumber  = indexOfHeaderMagic + lengthOfHeaderMagic
	lengthOfSequenceNumber = 4
	indexOfChannel         = indexOfSequenceNumber + lengthOfSequenceNumber
)

// size limits.
const (
	// lengthOfLargestPayload is the length in bytes of the largest (non-fragmented) LCM datagram.
	//
	// A small message is defined as one where the LCM small message header,
	// channel name, and the payload can fit into a single UDP datagram.
	// While this can technically be up to 64 kb, in practice the current
	// LCM implementations limit this to 1400 bytes (to stay under the Ethernet MTU).
	lengthOfLargestPayload  = 65499
	lengthOfSmallestMessage = indexOfChannel + 1
	lengthOfLongestChannel  = 63 // 64 including null byte
)

// headerMagic is the uint32 magic number signifying a short LCM message.
const headerMagic = 0x4c433032

// Message represents an LCM message.
type Message struct {
	Channel        string
	SequenceNumber uint32
	Data           []byte
}

// marshal an LCM message.
func (m *Message) marshal(b []byte) (int, error) {
	if len(m.Channel) > lengthOfLongestChannel {
		return 0, xerrors.Errorf("channel too long: %v bytes", len(m.Channel))
	}
	payloadSize := len(m.Channel) + 1 + len(m.Data)
	if payloadSize > lengthOfLargestPayload {
		return 0, xerrors.Errorf("channel and data too long: %v bytes", payloadSize)
	}
	binary.BigEndian.PutUint32(b[indexOfHeaderMagic:], headerMagic)
	binary.BigEndian.PutUint32(b[indexOfSequenceNumber:], m.SequenceNumber)
	copy(b[indexOfChannel:], m.Channel)
	b[indexOfChannel+len(m.Channel)] = 0
	copy(b[indexOfChannel+len(m.Channel)+1:], m.Data)
	return lengthOfHeaderMagic + lengthOfSequenceNumber + payloadSize, nil
}

// unmarshal an LCM message.
func (m *Message) unmarshal(data []byte) error {
	if len(data) < lengthOfSmallestMessage {
		return xerrors.Errorf("insufficient data: %v bytes", len(data))
	}
	header := binary.BigEndian.Uint32(data[indexOfHeaderMagic:])
	if header != headerMagic {
		return xerrors.Errorf("wrong header magic: 0x%x", header)
	}
	sequence := binary.BigEndian.Uint32(data[indexOfSequenceNumber:])
	offsetOfNullByte := bytes.IndexByte(data[indexOfChannel:], 0)
	if offsetOfNullByte == -1 {
		return xerrors.New("invalid channel: not null-terminated")
	}
	indexOfPayload := indexOfChannel + offsetOfNullByte + 1
	m.Channel = string(data[indexOfChannel : indexOfPayload-1])
	m.SequenceNumber = sequence
	m.Data = data[indexOfPayload:]
	return nil
}
