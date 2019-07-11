package lcm

import (
	"bytes"
	"encoding/binary"

	"golang.org/x/xerrors"
)

// field fixed lengths.
const (
	shortHeaderMagicSize    = 4
	shortHeaderSequenceSize = 4
	shortHeaderSize         = shortHeaderMagicSize + shortHeaderSequenceSize
)

// field max lengths.
const (
	// shortMessageMaxSize is the maximum size of a small (non-fragmented) LCM datagram.
	//
	// A small message is defined as one where the LCM small message header,
	// channel name, and the payload can fit into a single UDP datagram.
	// While this can technically be up to 64 kb, in practice the current
	// LCM implementations limit this to 1400 bytes (to stay under the Ethernet MTU).
	shortMessageMaxSize = 65499

	// maxChannelNameLength is the longest allowed channel name counted in bytes.
	maxChannelNameLength = 63 // 64 including null byte
)

// shortHeaderMagic is the uint32 magic number signifying a short LCM message.
const shortHeaderMagic = 0x4c433032

// field start indices.
const (
	indexOfShortHeaderMagic    = 0
	indexOfShortHeaderSequence = indexOfShortHeaderMagic + shortHeaderMagicSize
	indexOfChannelName         = indexOfShortHeaderSequence + shortHeaderSequenceSize
)

// Message represents an LCM message.
type Message struct {
	Channel        string
	SequenceNumber uint32
	Data           []byte
}

// Unmarshal an LCM message.
func (m *Message) Unmarshal(data []byte) error {
	if len(data) < 8 {
		return xerrors.Errorf("to small to be an LCM message: %v", len(data))
	}
	header := binary.BigEndian.Uint32(data[indexOfShortHeaderMagic:shortHeaderMagicSize])
	if header != shortHeaderMagic {
		return xerrors.Errorf("invalid header magic: %v", header)
	}
	sequence := binary.BigEndian.Uint32(data[indexOfShortHeaderSequence:shortHeaderSize])
	i := bytes.IndexByte(data[indexOfChannelName:], 0)
	if i == -1 {
		return xerrors.New("invalid format for channel name, couldn't find string-termination")
	}
	indexOfPayload := i + indexOfChannelName + 1
	m.Channel = string(data[indexOfChannelName : indexOfPayload-1])
	m.SequenceNumber = sequence
	m.Data = data[indexOfPayload:]
	return nil
}
