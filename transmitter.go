package lcm

import (
	"context"
	"net"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/ipv4"
	"golang.org/x/xerrors"
)

// Transmitter represents an LCM Transmitter instance.
type Transmitter struct {
	opts           *transmitterOptions
	conn           *ipv4.PacketConn
	sequenceNumber uint32
	messageBuf     []ipv4.Message
	payloadBuf     [lengthOfLargestUDPMessage]byte
	protoBuf       proto.Buffer
	msg            Message
}

// DialMulticastUDP returns a Transmitter configured with the provided options.
func DialMulticastUDP(ctx context.Context, transmitterOpts ...TransmitterOption) (*Transmitter, error) {
	opts := defaultTransmitterOptions()
	for _, transmitterOpt := range transmitterOpts {
		transmitterOpt(opts)
	}
	var listenConfig net.ListenConfig
	c, err := listenConfig.ListenPacket(ctx, "udp4", "")
	if err != nil {
		return nil, xerrors.Errorf("dial multicast UDP: %w", err)
	}
	udpConn := c.(*net.UDPConn)
	conn := ipv4.NewPacketConn(udpConn)
	if err := conn.SetMulticastTTL(opts.ttl); err != nil {
		return nil, xerrors.Errorf("dial multicast UDP: %w", err)
	}
	if opts.interfaceName != "" {
		ifi, err := net.InterfaceByName(opts.interfaceName)
		if err != nil {
			return nil, xerrors.Errorf("dial multicast UDP: %w", err)
		}
		if err := conn.SetMulticastInterface(ifi); err != nil {
			return nil, xerrors.Errorf("dial multicast UDP: %w", err)
		}
	}
	if err := conn.SetMulticastLoopback(opts.loopback); err != nil {
		return nil, xerrors.Errorf("dial multicast UDP: %w", err)
	}
	tx := &Transmitter{opts: opts, conn: conn}
	if len(opts.addrs) == 0 {
		opts.addrs = append(opts.addrs, &net.UDPAddr{IP: DefaultMulticastIP(), Port: DefaultPort})
	}
	for _, addr := range opts.addrs {
		tx.messageBuf = append(tx.messageBuf, ipv4.Message{
			Buffers: [][]byte{nil},
			Addr:    addr,
		})
	}
	return tx, nil
}

// TransmitProto transmits a protobuf message.
func (t *Transmitter) TransmitProto(ctx context.Context, channel string, m proto.Message) error {
	t.protoBuf.Reset()
	if err := t.protoBuf.Marshal(m); err != nil {
		return xerrors.Errorf("transmit proto: %w", err)
	}
	return t.Transmit(ctx, channel, t.protoBuf.Bytes())
}

// Transmit a raw payload.
//
// If the provided context has a deadline, it will be propagated to the underlying write operation.
func (t *Transmitter) Transmit(ctx context.Context, channel string, data []byte) error {
	t.msg.Data = data
	t.msg.Channel = channel
	t.msg.SequenceNumber = t.sequenceNumber
	t.sequenceNumber++
	n, err := t.msg.marshal(t.payloadBuf[:])
	if err != nil {
		return xerrors.Errorf("transmit to LCM: %w", err)
	}
	for i := range t.messageBuf {
		t.messageBuf[i].Buffers[0] = t.payloadBuf[:n]
		t.messageBuf[i].N = n
	}
	deadline, _ := ctx.Deadline()
	if err := t.conn.SetWriteDeadline(deadline); err != nil {
		return xerrors.Errorf("transmit to LCM: %w", err)
	}
	var transmitCount int
	for transmitCount < len(t.messageBuf) {
		n, err := t.conn.WriteBatch(t.messageBuf[transmitCount:], 0)
		if err != nil {
			return xerrors.Errorf("transmit to LCM: %w", err)
		}
		transmitCount += n
	}
	return nil
}

// Close the transmitter connection.
func (t *Transmitter) Close() error {
	if err := t.conn.Close(); err != nil {
		return xerrors.Errorf("close LCM transmitter: %w", err)
	}
	return nil
}
