package lcm

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"runtime"
	"strings"

	"github.com/einride/lcm-go/pkg/lz4"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/bpf"
	"golang.org/x/net/ipv4"
)

type Decompressor interface {
	Decompress(data []byte) ([]byte, error)
}

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
		return nil, fmt.Errorf("listen LCM UDP multicast: %w", err)
	}
	udpConn := packetConn.(*net.UDPConn)
	if err := udpConn.SetReadBuffer(opts.bufferSizeBytes); err != nil {
		return nil, fmt.Errorf("listen LCM UDP multicast: %w", err)
	}
	conn := ipv4.NewPacketConn(udpConn)
	if len(opts.ips) == 0 {
		opts.ips = append(opts.ips, DefaultMulticastIP())
	}
	rx := &Receiver{
		conn:          conn,
		opts:          opts,
		protoMessages: make(map[string]proto.Message),
		decompressors: map[string]Decompressor{"z=lz4": lz4.NewDecompressor()},
	}
	if opts.interfaceName != "" {
		ifi, err := net.InterfaceByName(opts.interfaceName)
		if err != nil {
			return nil, fmt.Errorf("listen LCM UDP multicast: interface %s: %w", opts.interfaceName, err)
		}
		if ifi.Flags&net.FlagMulticast == 0 {
			return nil, fmt.Errorf("listen LCM UDP multicast: interface %s: not a multicast interface", ifi.Name)
		}
		if ifi.Flags&net.FlagUp == 0 {
			return nil, fmt.Errorf("listen LCM UDP multicast: interface %s: not up", ifi.Name)
		}
		rx.ifi = ifi
	}
	for _, ip := range opts.ips {
		// from: https://godoc.org/golang.org/x/net/ipv4#hdr-Multicasting
		//
		// Note that the service port for transport layer protocol does not matter with this operation as joining
		// groups affects only network and link layer protocols, such as IPv4 and Ethernet.
		if err := conn.JoinGroup(rx.ifi, &net.UDPAddr{IP: ip}); err != nil {
			return nil, fmt.Errorf("listen LCM UDP multicast: IP %v: %w", ip, err)
		}
	}
	// contralFlags are the control flags used to configure the LCM connection.
	const controlFlags = ipv4.FlagInterface | ipv4.FlagDst | ipv4.FlagSrc
	if err := conn.SetControlMessage(controlFlags, true); err != nil {
		return nil, fmt.Errorf("listen LCM UDP multicast: %w", err)
	}
	if runtime.GOOS == "linux" {
		if len(opts.bpfProgram) > 0 {
			rawBPFInstructions, err := bpf.Assemble(opts.bpfProgram)
			if err != nil {
				return nil, fmt.Errorf("listen LCM UDP multicast: %w", err)
			}
			if err := conn.SetBPF(rawBPFInstructions); err != nil {
				return nil, fmt.Errorf("listen LCM UDP multicast: %w", err)
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
	protoMessages   map[string]proto.Message
	protoMessage    proto.Message
	decompressors   map[string]Decompressor
}

// Receive an LCM message.
//
// If the provided context has a deadline, it will be propagated to the underlying read operation.
func (r *Receiver) Receive(ctx context.Context) error {
	r.protoMessage = nil
	if r.messageBufIndex >= r.messageBufSize {
		r.messageBufIndex = 0
		deadline, _ := ctx.Deadline()
		if err := r.conn.SetReadDeadline(deadline); err != nil {
			return fmt.Errorf("receive on LCM: %w", err)
		}
		n, err := r.conn.ReadBatch(r.messageBuf, 0)
		if err != nil {
			return fmt.Errorf("receive on LCM: %w", err)
		}
		r.messageBufSize = n
	}
	curr := r.messageBuf[r.messageBufIndex]
	r.messageBufIndex++
	var cm ipv4.ControlMessage
	if err := cm.Parse(curr.OOB[:curr.NN]); err != nil {
		return fmt.Errorf("receive on LCM: %w", err)
	}
	r.srcAddr = cm.Src
	r.dstAddr = cm.Dst
	r.ifIndex = cm.IfIndex
	if err := r.currMessage.unmarshal(curr.Buffers[0][:curr.N]); err != nil {
		return fmt.Errorf("receive on LCM: %w", err)
	}
	params := strings.Split(r.currMessage.Params, "&")
	if len(params) > 1 {
		return fmt.Errorf("receive multiple query params not supported")
	}
	if decompressor, ok := r.decompressors[params[0]]; ok {
		data, err := decompressor.Decompress(r.currMessage.Data)
		if err != nil {
			return fmt.Errorf("decompressor on LCM: %w", err)
		}
		r.currMessage.Data = data
	}
	return nil
}

// Receive a proto LCM message. The channel is assumed to be a fully-qualified message name.
func (r *Receiver) ReceiveProto(ctx context.Context) error {
	if err := r.Receive(ctx); err != nil {
		return err
	}
	protoMessage, ok := r.protoMessages[r.currMessage.Channel]
	if !ok {
		messageType := proto.MessageType(r.currMessage.Channel)
		if messageType == nil {
			return nil // don't error on encountering non-proto channels
		}
		protoMessage = reflect.New(messageType.Elem()).Interface().(proto.Message)
		r.protoMessages[r.currMessage.Channel] = protoMessage
	}
	if err := proto.Unmarshal(r.currMessage.Data, protoMessage); err != nil {
		return fmt.Errorf("receive proto %s on LCM: %w", r.currMessage.Channel, err)
	}
	r.protoMessage = protoMessage
	return nil
}

// ProtoMessage returns the last received proto message.
func (r *Receiver) ProtoMessage() proto.Message {
	return r.protoMessage
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
			return fmt.Errorf("close LCM receiver: %w", err)
		}
	}
	return r.conn.Close()
}
