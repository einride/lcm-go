package lcm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
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

// shortMessageMagic is the uint32 magic number signifying a short LCM message.
const (
	shortMessageMagic uint32 = 0x4c433032
)

// Message represents an LCM message.
type Message struct {
	Channel        string
	Params         string
	SequenceNumber uint32
	Data           []byte
}

// marshal an LCM message.
func (m *Message) marshal(b []byte) (int, error) {
	rawChannel := m.Channel
	if m.Params != "" {
		rawChannel += "?" + m.Params
	}
	cLen := len(rawChannel)
	if cLen > lengthOfLongestChannel {
		return 0, fmt.Errorf("channel too long: %v bytes", len(m.Channel))
	}
	payloadSize := cLen + 1 + len(m.Data)
	if payloadSize > lengthOfLargestPayload {
		return 0, fmt.Errorf("channel and data too long: %v bytes", payloadSize)
	}
	binary.BigEndian.PutUint32(b[indexOfHeaderMagic:], shortMessageMagic)
	binary.BigEndian.PutUint32(b[indexOfSequenceNumber:], m.SequenceNumber)
	copy(b[indexOfChannel:], rawChannel)
	b[indexOfChannel+cLen] = 0
	copy(b[indexOfChannel+cLen+1:], m.Data)
	return lengthOfHeaderMagic + lengthOfSequenceNumber + payloadSize, nil
}

func split(s string, c byte) (string, string) {
	i := strings.IndexByte(s, c)
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+1:]
}

// unmarshal an LCM message.
func (m *Message) unmarshal(data []byte) error {
	if len(data) < lengthOfSmallestMessage {
		return fmt.Errorf("insufficient data: %v bytes", len(data))
	}
	header := binary.BigEndian.Uint32(data[indexOfHeaderMagic:])
	if header != shortMessageMagic {
		return fmt.Errorf("wrong header magic: 0x%x", header)
	}
	sequence := binary.BigEndian.Uint32(data[indexOfSequenceNumber:])
	offsetOfNullByte := bytes.IndexByte(data[indexOfChannel:], 0)
	if offsetOfNullByte == -1 {
		return errors.New("invalid channel: not null-terminated")
	}
	indexOfPayload := indexOfChannel + offsetOfNullByte + 1
	m.Channel, m.Params = split(string(data[indexOfChannel:indexOfPayload-1]), '?')
	m.SequenceNumber = sequence
	m.Data = data[indexOfPayload:]
	return nil
}
