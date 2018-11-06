package lcm

import (
	"net/url"
	"strconv"

	"github.com/pkg/errors"
)

// url constants.
const (
	schemeUDPM              = "udpm"
	optionTTL               = "ttl"
	optionReceiveBufferSize = "recv_buf_size"
)

// Provider represents an LCM provider specification.
type Provider interface {
	// URL returns an URL representation of the Provider.
	URL() *url.URL
}

// ParseProvider parses an LCM provider specification.
func ParseProvider(uri string) (Provider, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, errors.Wrap(err, "invalid provider URI")
	}
	switch u.Scheme {
	case schemeUDPM:
		p := &UDPMulticastProvider{
			Address: u.Host,
		}
		if u.Query().Get(optionTTL) != "" {
			ttl, err := strconv.Atoi(u.Query().Get(optionTTL))
			if err != nil {
				return nil, errors.Wrapf(err, "invalid option: %v", optionTTL)
			}
			p.TTL = ttl
		}
		if u.Query().Get(optionReceiveBufferSize) != "" {
			receiveBufferSize, err := strconv.Atoi(u.Query().Get(optionReceiveBufferSize))
			if err != nil {
				return nil, errors.Wrapf(err, "invalid option: %v", optionReceiveBufferSize)
			}
			p.ReceiveBufferSize = receiveBufferSize
		}
		return p, nil
	default:
		return nil, errors.Errorf("unsupported provider URI scheme: %v", u.Scheme)
	}
}

// UDPMulticastProvider represents an UDP multicast LCM provider specification.
type UDPMulticastProvider struct {
	Address           string
	TTL               int
	ReceiveBufferSize int
}

// URL returns an URL representation of the Provider.
func (p *UDPMulticastProvider) URL() *url.URL {
	u := &url.URL{
		Scheme: schemeUDPM,
		Host:   p.Address,
	}
	if p.TTL > 0 {
		u.Query().Set(optionTTL, strconv.Itoa(p.TTL))
	}
	if p.ReceiveBufferSize > 0 {
		u.Query().Set(optionReceiveBufferSize, strconv.Itoa(p.ReceiveBufferSize))
	}
	return u
}
