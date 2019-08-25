package lcm

import "net"

// transmitterOptions are the configuration options for an LCM transmitter.
type transmitterOptions struct {
	ttl           int
	loopback      bool
	interfaceName string
	addrs         []*net.UDPAddr
}

// defaultTransmitterOptions returns transmitter options with sensible default values.
func defaultTransmitterOptions() *transmitterOptions {
	return &transmitterOptions{
		loopback: true,
		ttl:      1,
	}
}

// TransmitterOption configures an LCM transmitter.
type TransmitterOption func(*transmitterOptions)

// WithTransmitterInterface configures the interface to transmit on.
func WithTransmitterInterface(interfaceName string) TransmitterOption {
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

// WithMulticastLoopback configures multicast loopback on the transmitter socket.
func WithMulticastLoopback(b bool) TransmitterOption {
	return func(opts *transmitterOptions) {
		opts.loopback = b
	}
}

// WithTTL configures the multicast TTL on the transmitter socket.
func WithTTL(ttl int) TransmitterOption {
	return func(opts *transmitterOptions) {
		opts.ttl = ttl
	}
}
