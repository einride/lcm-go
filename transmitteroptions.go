package lcm

import (
	"github.com/golang/protobuf/proto"
	"net"
)

// transmitterOptions are the configuration options for an LCM transmitter.
type transmitterOptions struct {
	ttl           int
	loopback      bool
	compressor    map[string]Compressor
	interfaceName string
	addrs         []*net.UDPAddr
}

// defaultTransmitterOptions returns transmitter options with sensible default values.
func defaultTransmitterOptions() *transmitterOptions {
	return &transmitterOptions{
		loopback:   true,
		ttl:        1,
		compressor: make(map[string]Compressor),
	}
}

// TransmitterOption configures an LCM transmitter.
type TransmitterOption func(*transmitterOptions)

// WithTransmitInterface configures the interface to transmit on.
func WithTransmitInterface(interfaceName string) TransmitterOption {
	return func(opts *transmitterOptions) {
		opts.interfaceName = interfaceName
	}
}

// WithTransmitAddress configures an address to transmit to.
//
// Provide this option multiple times to transmit to multiple addresses.
func WithTransmitAddress(addr *net.UDPAddr) TransmitterOption {
	return func(opts *transmitterOptions) {
		opts.addrs = append(opts.addrs, addr)
	}
}

// WithTransmitMulticastLoopback configures multicast loopback on the transmitter socket.
func WithTransmitMulticastLoopback(b bool) TransmitterOption {
	return func(opts *transmitterOptions) {
		opts.loopback = b
	}
}

// WithTransmitCompression configures compressor on a certain channel
func WithTransmitCompression(compressor Compressor, msgs... proto.Message) TransmitterOption {
	return func(opts *transmitterOptions) {
		for _, msg := range msgs {
			opts.compressor[proto.MessageName(msg)] = compressor
		}
	}
}

// WithTransmitTTL configures the multicast TTL on the transmitter socket.
func WithTransmitTTL(ttl int) TransmitterOption {
	return func(opts *transmitterOptions) {
		opts.ttl = ttl
	}
}
