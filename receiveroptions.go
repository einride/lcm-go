package lcm

import (
	"net"

	"golang.org/x/net/bpf"
	"google.golang.org/protobuf/proto"
)

// receiverOptions are the configuration options for an LCM receiver.
type receiverOptions struct {
	interfaceName   string
	port            int
	ips             []net.IP
	bufferSizeBytes int
	batchSize       int
	bpfProgram      []bpf.Instruction
	protos          []proto.Message
}

// DefaultMulticastIP returns the default LCM multicast IP.
func DefaultMulticastIP() net.IP {
	return net.IPv4(239, 255, 76, 67)
}

// DefaultPort is the default LCM port.
const DefaultPort = 7667

// defaultReceiverOptions returns receiver options with sensible default values.
func defaultReceiverOptions() *receiverOptions {
	return &receiverOptions{
		batchSize:       5,
		port:            DefaultPort,
		bufferSizeBytes: 2097152,              // 2MB (from the LCM documentation)
		bpfProgram:      shortMessageFilter(), // TODO: add support for fragmented messages
	}
}

// ReceiverOption configures an LCM receiver.
type ReceiverOption func(*receiverOptions)

// WithReceivePort configures the port to listen on.
func WithReceivePort(port int) ReceiverOption {
	return func(o *receiverOptions) {
		o.port = port
	}
}

// WithReceiveInterface configures the interface to receive on.
func WithReceiveInterface(interfaceName string) ReceiverOption {
	return func(o *receiverOptions) {
		o.interfaceName = interfaceName
	}
}

// WithReceiveAddress a multicast group address to receive from.
//
// Provide this option multiple times to join multiple multicast groups.
func WithReceiveAddress(ip net.IP) ReceiverOption {
	return func(o *receiverOptions) {
		o.ips = append(o.ips, ip)
	}
}

// WithReceiveBPF configures the Berkely Packet Filter to set on the receiver socket.
//
// Ineffectual in non-Linux environments.
func WithReceiveBPF(program []bpf.Instruction) ReceiverOption {
	return func(o *receiverOptions) {
		o.bpfProgram = program
	}
}

func WithReceiveProtos(msgs ...proto.Message) ReceiverOption {
	return func(o *receiverOptions) {
		o.bpfProgram = shortProtoMessageFilter(msgs...)
		o.protos = msgs
	}
}

// WithReceiveBufferSize configures the kernel read buffer size (in bytes).
func WithReceiveBufferSize(n int) ReceiverOption {
	return func(o *receiverOptions) {
		o.bufferSizeBytes = n
	}
}

// WithReceiveBatchSize configures the max number of messages to receive from the kernel in a single batch.
func WithReceiveBatchSize(n int) ReceiverOption {
	return func(o *receiverOptions) {
		o.batchSize = n
	}
}
