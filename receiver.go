package lcm

import (
	"context"
	"net"
	"time"

	"golang.org/x/xerrors"
)

// UDPReader is an interface for a connection that a Receiver can read UDP messages from.
type UDPReader interface {
	ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
	SetReadDeadline(time.Time) error
	Close() error
}

// Receiver represents an LCM Receiver instance.
//
// Not thread-safe.
type Receiver struct {
	r       UDPReader
	buf     [lengthOfLargestUDPMessage]byte
	message Message
}

// NewReceiver creates a listener instance.
func NewReceiver(reader UDPReader) *Receiver {
	return &Receiver{r: reader}
}

// Receive an LCM message.
//
// If the provided context has a deadline, it will be propagated to the underlying read operation.
func (r *Receiver) Receive(ctx context.Context) error {
	deadline, _ := ctx.Deadline()
	if err := r.r.SetReadDeadline(deadline); err != nil {
		return xerrors.Errorf("receive: %w", err)
	}
	n, _, err := r.r.ReadFromUDP(r.buf[:])
	if err != nil {
		return xerrors.Errorf("receive: %w", err)
	}
	return r.message.unmarshal(r.buf[:n])
}

// Message returns the last received message.
func (r *Receiver) Message() *Message {
	return &r.message
}

// Close the receiver connection.
func (r *Receiver) Close() error {
	return r.r.Close()
}
