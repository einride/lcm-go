package lcm

import (
	"context"
	"fmt"
	"net"
	"runtime"

	"golang.org/x/net/bpf"
	"golang.org/x/net/ipv4"
	"golang.org/x/xerrors"
)

// ListenMulticastUDP returns a Receiver configured with the provided options.
func ListenMulticastUDP(ctx context.Context, receiverOpts ...ReceiverOption) (*Receiver, error) {
	opts := defaultReceiverOptions()
	for _, receiverOpt := range receiverOpts {
		receiverOpt(opts)
	}
	var listenConfig net.ListenConfig
	// wildcard address prefix for all administratively-scoped (local) multicast addresses
	packetConn, err := listenConfig.ListenPacket(ctx, "udp4", fmt.Sprintf("239.0.0.0:%d", opts.port))
	if err != nil {
		return nil, xerrors.Errorf("listen LCM UDP multicast: %w", err)
	}
	udpConn := packetConn.(*net.UDPConn)
	if err := udpConn.SetReadBuffer(opts.bufferSizeBytes); err != nil {
		return nil, xerrors.Errorf("listen LCM UDP multicast: %w", err)
	}
	conn := ipv4.NewPacketConn(udpConn)
	if len(opts.ips) == 0 {
		opts.ips = append(opts.ips, DefaultMulticastIP())
	}
	rx := &Receiver{conn: conn, opts: opts}
	if opts.interfaceName != "" {
		ifi, err := net.InterfaceByName(opts.interfaceName)
		if err != nil {
			return nil, xerrors.Errorf("listen LCM UDP multicast: interface %s: %w", opts.interfaceName, err)
		}
		if ifi.Flags&net.FlagMulticast == 0 {
			return nil, xerrors.Errorf("listen LCM UDP multicast: interface %s: not a multicast interface", ifi.Name)
		}
		if ifi.Flags&net.FlagUp == 0 {
			return nil, xerrors.Errorf("listen LCM UDP multicast: interface %s: not up", ifi.Name)
		}
		rx.ifi = ifi
	}
	for _, ip := range opts.ips {
		// from: https://godoc.org/golang.org/x/net/ipv4#hdr-Multicasting
		//
		// Note that the service port for transport layer protocol does not matter with this operation as joining
		// groups affects only network and link layer protocols, such as IPv4 and Ethernet.
		if err := conn.JoinGroup(rx.ifi, &net.UDPAddr{IP: ip}); err != nil {
			return nil, xerrors.Errorf("listen LCM UDP multicast: IP %v: %w", ip, err)
		}
	}
	// contralFlags are the control flags used to configure the LCM connection.
	const controlFlags = ipv4.FlagInterface | ipv4.FlagDst | ipv4.FlagSrc
	if err := conn.SetControlMessage(controlFlags, true); err != nil {
		return nil, xerrors.Errorf("listen LCM UDP multicast: %w", err)
	}
	if runtime.GOOS == "linux" {
		if len(opts.bpfProgram) > 0 {
			rawBPFInstructions, err := bpf.Assemble(opts.bpfProgram)
			if err != nil {
				return nil, xerrors.Errorf("listen LCM UDP multicast: %w", err)
			}
			if err := conn.SetBPF(rawBPFInstructions); err != nil {
				return nil, xerrors.Errorf("listen LCM UDP multicast: %w", err)
			}
		}
	}
	// allocate memory for batch reads
	for i := 0; i < opts.batchSize; i++ {
		rx.messageBuf = append(rx.messageBuf, ipv4.Message{
			Buffers: [][]byte{
				make([]byte, lengthOfLargestUDPMessage),
			},
			OOB: ipv4.NewControlMessage(controlFlags),
		})
	}
	return rx, nil
}

// Receiver represents an LCM Receiver instance.
//
// Not thread-safe.
type Receiver struct {
	opts            *receiverOptions
	conn            *ipv4.PacketConn
	ifi             *net.Interface
	messageBuf      []ipv4.Message
	messageBufSize  int
	messageBufIndex int
	currMessage     Message
	dstAddr         net.IP
	srcAddr         net.IP
	ifIndex         int
}

// Receive an LCM message.
//
// If the provided context has a deadline, it will be propagated to the underlying read operation.
func (r *Receiver) Receive(ctx context.Context) error {
	if r.messageBufIndex >= r.messageBufSize {
		r.messageBufIndex = 0
		deadline, _ := ctx.Deadline()
		if err := r.conn.SetReadDeadline(deadline); err != nil {
			return xerrors.Errorf("receive on LCM: %w", err)
		}
		n, err := r.conn.ReadBatch(r.messageBuf, 0)
		if err != nil {
			return xerrors.Errorf("receive on LCM: %w", err)
		}
		r.messageBufSize = n
	}
	curr := r.messageBuf[r.messageBufIndex]
	r.messageBufIndex++
	var cm ipv4.ControlMessage
	if err := cm.Parse(curr.OOB[:curr.NN]); err != nil {
		return xerrors.Errorf("receive on LCM: %w", err)
	}
	r.srcAddr = cm.Src
	r.dstAddr = cm.Dst
	r.ifIndex = cm.IfIndex
	if err := r.currMessage.unmarshal(curr.Buffers[0][:curr.N]); err != nil {
		return xerrors.Errorf("receive on LCM: %w", err)
	}
	return nil
}

// Message returns the last received message.
func (r *Receiver) Message() *Message {
	return &r.currMessage
}

// SourceAddress returns the source address of the last received message.
func (r *Receiver) SourceAddress() net.IP {
	return r.srcAddr
}

// DestinationAddress returns the destination address of the last received message.
func (r *Receiver) DestinationAddress() net.IP {
	return r.dstAddr
}

// InterfaceIndex returns the interface index of the last received message.
func (r *Receiver) InterfaceIndex() int {
	return r.ifIndex
}

// Close the receiver connection after leaving all joined multicast groups.
func (r *Receiver) Close() error {
	for _, ip := range r.opts.ips {
		if err := r.conn.LeaveGroup(r.ifi, &net.UDPAddr{IP: ip}); err != nil {
			return xerrors.Errorf("close LCM receiver: %w", err)
		}
	}
	return r.conn.Close()
}
