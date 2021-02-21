package lcmlog

import (
	"bufio"
	"fmt"
	"io"
	"os"
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
	s.msg.unmarshalBinary(s.sc.Bytes())
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

func (s *Scanner) SplitWrite(fileName string, splitSizeMByte uint32) error {
	var logWriter io.WriteCloser
	var logCounter uint32
	var bytesWritten uint64
	for s.sc.Scan() {
		// Close log if reached max split size
		if splitSizeMByte > 0 && bytesWritten >= uint64(splitSizeMByte*1000000) {
			if err := logWriter.Close(); err != nil {
				return fmt.Errorf("close log writer: %w", err)
			}
			bytesWritten = 0
		}
		// Open new log
		if bytesWritten == 0 {
			out, err := os.Create(fileName + fmt.Sprintf(".%d", logCounter))
			if err != nil {
				return fmt.Errorf("init log writer: %w", err)
			}
			logWriter = out
			logCounter++
		}
		// Write data to the open log
		n, err := logWriter.Write(s.sc.Bytes())
		if err != nil {
			return fmt.Errorf("write to log file: %w", err)
		}
		bytesWritten += uint64(n)
	}
	// Close the remaining (last) log
	if bytesWritten != 0 {
		if err := logWriter.Close(); err != nil {
			return fmt.Errorf("close log writer: %w", err)
		}
	}
	return nil
}
