package lcm

import (
	"context"
	"time"

	"golang.org/x/xerrors"
)

// UDPWriter is an interface for a connection that an LCM Transmitter can write messages to.
type UDPWriter interface {
	Write([]byte) (int, error)
	SetWriteDeadline(time.Time) error
	Close() error
}

// Transmitter represents an LCM Transmitter instance.
type Transmitter struct {
	w              UDPWriter
	sequenceNumber uint32
	buf            [lengthOfLargestUDPMessage]byte
	msg            Message
}

// NewTransmitter creates a new LCM transmitter.
func NewTransmitter(w UDPWriter) *Transmitter {
	return &Transmitter{w: w}
}

// Transmit an LCM message.
//
// If the provided context has a deadline, it will be propagated to the underlying write operation.
func (t *Transmitter) Transmit(ctx context.Context, channel string, data []byte) error {
	t.msg.Data = data
	t.msg.Channel = channel
	t.msg.SequenceNumber = t.sequenceNumber
	t.sequenceNumber++
	n, err := t.msg.marshal(t.buf[:])
	if err != nil {
		return xerrors.Errorf("transmit: %w", err)
	}
	deadline, _ := ctx.Deadline()
	if err := t.w.SetWriteDeadline(deadline); err != nil {
		return xerrors.Errorf("transmit: %w", err)
	}
	if _, err := t.w.Write(t.buf[:n]); err != nil {
		return xerrors.Errorf("transmit: %w", err)
	}
	return nil
}

// Close the transmitter connection.
func (t *Transmitter) Close() error {
	return t.w.Close()
}
