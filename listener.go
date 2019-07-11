package lcm

import (
	"net"
	"time"

	"golang.org/x/xerrors"
)

// Listener represents an LCM Listener instance.
// Not thread-safe.
type Listener struct {
	conn *net.UDPConn
	data []byte
}

// NewListener creates a listener instance.
func NewListener(addr *net.UDPAddr) (*Listener, error) {
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return nil, xerrors.Errorf("new listener: %w", err)
	}
	return &Listener{
		conn: conn,
		data: make([]byte, shortMessageMaxSize+shortHeaderSize),
	}, nil
}

// Receive reads data from the listener and populates the message provided.
func (l *Listener) Receive(m *Message) error {
	n, err := l.conn.Read(l.data)
	if err != nil {
		return xerrors.Errorf("receive: %w", err)
	}
	return m.Unmarshal(l.data[:n])
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
