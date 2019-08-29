package lcmlog

import (
	"bufio"
	"io"
)

type Scanner struct {
	sc  *bufio.Scanner
	msg Message
}

func NewScanner(r io.Reader) *Scanner {
	sc := bufio.NewScanner(r)
	sc.Split(scanLogMessages)
	return &Scanner{sc: sc}
}

func (s *Scanner) Scan() bool {
	if !s.sc.Scan() {
		return false
	}
	s.msg.unmarshalBinary(s.sc.Bytes())
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
