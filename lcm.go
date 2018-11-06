// Package lcm provides Lightweight Communications and Marshalling primitives.
package lcm

import (
	"encoding/binary"
	"net"
	"sync"

	"github.com/pkg/errors"
)

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
	// shortHeaderSize is the number of bytes in the header of a short LCM message.
	shortHeaderSize = 8
)

// LCM represents an LCM instance.
type LCM struct {
	conn                  *net.UDPConn
	publishMutex          sync.Mutex
	publishSequenceNumber uint32
	publishBuffer         [shortHeaderSize + shortMessageMaxSize]byte
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

// Publish an LCM message.
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
	binary.BigEndian.PutUint32(lc.publishBuffer[0:], shortHeaderMagic)
	binary.BigEndian.PutUint32(lc.publishBuffer[4:], lc.publishSequenceNumber)
	lc.publishSequenceNumber++
	copy(lc.publishBuffer[shortHeaderSize:], []byte(channel))
	lc.publishBuffer[shortHeaderSize+channelSize] = 0
	copy(lc.publishBuffer[shortHeaderSize+channelSize+1:], data)
	packetSize := shortHeaderSize + payloadSize
	if _, err := lc.conn.Write(lc.publishBuffer[:packetSize]); err != nil {
		return errors.Wrap(err, "failed to publish")
	}
	return nil
}

// Close the LCM instance.
func (lc *LCM) Close() error {
	return lc.conn.Close()
}
