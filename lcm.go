// Package lcm provides Lightweight Communications and Marshalling primitives.
package lcm

import (
	"bytes"
	"encoding/binary"
	"net"
	"sync"
	"time"

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

// Transmitter represents an LCM Transmitter instance.
type Transmitter struct {
	conn                  *net.UDPConn
	publishMutex          sync.Mutex
	publishSequenceNumber uint32
	publishBuffer         [shortHeaderSize + shortMessageMaxSize]byte
}

// Listener represents an LCM Listener instance.
type Listener struct {
	conn *net.UDPConn
}

// Message represents an LCM message.
type Message struct {
	Channel        string
	SequenceNumber uint32
	Data           []byte
}

// NewTransmitter creates a transmitter instance.
func NewTransmitter(addr *net.UDPAddr) (*Transmitter, error) {
	if !addr.IP.IsMulticast() {
		return nil, xerrors.New("new transmitter: addr is not a multicast address")
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, xerrors.Errorf("new transmitter: %w", err)
	}
	return &Transmitter{conn: conn}, nil
}

// Publish an LCM message.
func (t *Transmitter) Publish(m *Message) error {
	channelSize := len(m.Channel)
	if channelSize > maxChannelNameLength {
		return xerrors.Errorf("channel name too big: %v bytes", channelSize)
	}
	payloadSize := channelSize + 1 + len(m.Data)
	if payloadSize > shortMessageMaxSize {
		return xerrors.Errorf("payload (channel + data) too big: %v bytes", payloadSize)
	}
	t.publishMutex.Lock()
	defer t.publishMutex.Unlock()
	binary.BigEndian.PutUint32(t.publishBuffer[indexOfShortHeaderMagic:], shortHeaderMagic)
	binary.BigEndian.PutUint32(t.publishBuffer[indexOfShortHeaderSequence:], t.publishSequenceNumber)
	t.publishSequenceNumber++
	copy(t.publishBuffer[indexOfChannelName:], []byte(m.Channel))
	t.publishBuffer[shortHeaderSize+channelSize] = 0
	copy(t.publishBuffer[indexOfChannelName+channelSize+1:], m.Data)
	packetSize := shortHeaderSize + payloadSize
	if _, err := t.conn.Write(t.publishBuffer[:packetSize]); err != nil {
		return xerrors.Errorf("publish: %w", err)
	}
	return nil
}

// SetWriteDeadline sets the write deadline for the transmitter.
func (t *Transmitter) SetWriteDeadline(time time.Time) error {
	if err := t.conn.SetWriteDeadline(time); err != nil {
		return xerrors.Errorf("set write deadline: %w", err)
	}
	return nil
}

// Close the transmitter connection.
func (t *Transmitter) Close() error {
	if err := t.conn.Close(); err != nil {
		return xerrors.Errorf("close: %w", err)
	}
	return nil
}

// NewListener creates a listener instance.
func NewListener(addr *net.UDPAddr) (*Listener, error) {
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return nil, xerrors.Errorf("new listener: %w", err)
	}
	return &Listener{conn: conn}, nil
}

// Receive reads data from the listener and populates the message provided.
func (l *Listener) Receive(m *Message) error {
	data := make([]byte, shortMessageMaxSize+shortHeaderSize)
	n, err := l.conn.Read(data)
	if err != nil {
		return xerrors.Errorf("receive: %w", err)
	}
	return m.Unmarshal(data[:n])
}

// SetReadDeadline sets the read deadline for the listener.
func (l *Listener) SetReadDeadline(time time.Time) error {
	if err := l.conn.SetReadDeadline(time); err != nil {
		return xerrors.Errorf("set read deadline: %w", err)
	}
	return nil
}

// Close the listener connection.
func (l *Listener) Close() error {
	if err := l.conn.Close(); err != nil {
		return xerrors.Errorf("close: %w", err)
	}
	return nil
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
