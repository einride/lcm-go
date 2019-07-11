package lcm

import (
	"encoding/binary"
	"net"
	"sync"
	"time"

	"golang.org/x/xerrors"
)

// Transmitter represents an LCM Transmitter instance.
type Transmitter struct {
	conn                  *net.UDPConn
	publishMutex          sync.Mutex
	publishSequenceNumber uint32
	publishBuffer         [shortHeaderSize + shortMessageMaxSize]byte
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
