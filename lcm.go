// Package LCM provides Lightweight Communications and Marshalling primitives.
package lcm

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"io"
	"net"
	"sync"
)

// field fixed lengths.
const (
	shortHeaderMagicSize = 4
	shortHeaderSequenceSize = 4
	shortHeaderSize = shortHeaderMagicSize + shortHeaderSequenceSize
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

const (
	// shortHeaderMagic is the uint32 magic number signifying a short LCM message.
	shortHeaderMagic = 0x4c433032
)

// field start indices.
const (
	indexOfShortHeaderMagic = 0
	indexOfShortHeaderSequence = indexOfShortHeaderMagic + shortHeaderMagicSize
	indexOfChannelName = indexOfShortHeaderSequence + shortHeaderSequenceSize
)

// Publisher represents an LCM Publisher instance.
type LCM struct {
	conn                  *net.UDPConn
	publishMutex          sync.Mutex
	publishSequenceNumber uint32
	publishBuffer         [shortHeaderSize + shortMessageMaxSize]byte
}

// Message represents a LCM message.
type Message struct {
	Topic	 string
	Sequence uint32
	Data     []byte
}

// Create an LCM instance.
func Create(provider string) (*LCM, error) {
	parsedProvider, err := ParseProvider(provider)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse provider")
	}
	switch p := parsedProvider.(type) {
	case *UDPMulticastProvider:
		addr, err := net.ResolveUDPAddr("udp", p.Address)
		if err != nil {
			return nil, errors.Wrap(err, "failed to resolve UDP address")
		}
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			return nil, errors.Wrap(err, "failed to dial UDP")
		}
		return &LCM{conn: conn}, nil
	default:
		return nil, errors.Errorf("unsupported provider: %v", provider)
	}
}

// Publish a LCM message.
func (lc *LCM) Publish(channel string, data []byte) error {
	channelSize := len(channel)
	if channelSize > maxChannelNameLength {
		return errors.Errorf("channel name too big: %v bytes", channelSize)
	}
	payloadSize := channelSize + 1 + len(data)
	if payloadSize > shortMessageMaxSize {
		return errors.Errorf("payload (channel + data) too big: %v bytes", payloadSize)
	}
	lc.publishMutex.Lock()
	defer lc.publishMutex.Unlock()
	binary.BigEndian.PutUint32(lc.publishBuffer[indexOfShortHeaderMagic:], shortHeaderMagic)
	binary.BigEndian.PutUint32(lc.publishBuffer[indexOfShortHeaderSequence:], lc.publishSequenceNumber)
	lc.publishSequenceNumber++
	copy(lc.publishBuffer[indexOfChannelName:], []byte(channel))
	lc.publishBuffer[shortHeaderSize+channelSize] = 0
	copy(lc.publishBuffer[indexOfChannelName+channelSize+1:], data)
	packetSize := shortHeaderSize + payloadSize
	if _, err := lc.conn.Write(lc.publishBuffer[:packetSize]); err != nil {
		return errors.Wrap(err, "failed to publish")
	}
	return nil
}

// Unmarshal a LCM message.
func (m *Message) Unmarshal(data []byte) error {
	if binary.BigEndian.Uint32(data[indexOfShortHeaderMagic:shortHeaderMagicSize]) != shortHeaderMagic {
		return errors.New("Not an LCM message")
	}
	sequence := binary.BigEndian.Uint32(data[indexOfShortHeaderSequence:shortHeaderSize])
	i := bytes.IndexRune(data[indexOfChannelName:], 0)
	if i == -1 {
		return errors.New("could not find channel name, i out of bounds")
	}
	indexOfPayload := i + indexOfChannelName + 1
	m.Topic = string(data[indexOfChannelName:indexOfPayload-1])
	m.Sequence = sequence
	m.Data = data[indexOfPayload:]
	return nil
}

// ReadMessage reads a LCM message.
func ReadMessage(r io.Reader)(*Message, error) {
	b := make([]byte, shortHeaderSize+shortMessageMaxSize)
	n, err := r.Read(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}
	var m Message
	return &m, (&m).Unmarshal(b[:n])
}

// Close the LCM instance.
func (lc *LCM) Close() error {
	return lc.conn.Close()
}
