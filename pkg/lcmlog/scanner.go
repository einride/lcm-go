package lcmlog

import (
	"bufio"
	"io"
	"strings"

	"github.com/einride/lcm-go/pkg/lz4"
)

type Decompressor interface {
	Decompress(data []byte) ([]byte, error)
}

type Scanner struct {
	sc            *bufio.Scanner
	msg           Message
	decompressors map[string]Decompressor
}

func NewScanner(r io.Reader) *Scanner {
	sc := bufio.NewScanner(r)
	sc.Split(scanLogMessages)
	return &Scanner{
		sc:            sc,
		decompressors: map[string]Decompressor{"z=lz4": lz4.NewDecompressor()},
	}
}

func (s *Scanner) Scan() bool {
	if !s.sc.Scan() {
		return false
	}
	s.msg.UnmarshalBinary(s.sc.Bytes())
	params := strings.Split(s.msg.Params, "&")
	if decompressor, ok := s.decompressors[params[0]]; ok {
		data, err := decompressor.Decompress(s.msg.Data)
		if err != nil {
			return false
		}
		s.msg.Data = data
	}
	return true
}

func (s *Scanner) RawMessage() []byte {
	return s.sc.Bytes()
}

func (s *Scanner) Message() *Message {
	return &s.msg
}

func (s *Scanner) Err() error {
	return s.sc.Err()
}
