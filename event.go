package lcm

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"

	"golang.org/x/xerrors"
)

// lcmSyncWord is the uint32 integer signifying an LCM log event.
const lcmSyncWord = 0xEDA1DA01

// LogReader represents an LCM log file reader
type LogReader struct {
	event        Event
	protoMessage proto.Message
	logFile      *os.File
}

// Event represents one event in the ordered list of events in an LCM log file.
//
// The header is followed by the UTF-8 encoding of the LCM channel. The channel is NOT NULL-terminated.
// The channel is followed by the message data.
// All data is packed in big endian order.
type Event struct {
	EventHeader EventHeader
	Channel     string
	Params      string
	Data        []byte
}

// EventHeader represents the first 28 bytes of an LCM event.
//
// EventNumber is a monotonically increasing uint64 starting at 0 and increases by 1 for each event.
// Timestamp is a monotonically increasing uint64 with the number of microseconds since
// 00:00:00 UTC on January 1, 1970.
// ChannelLength is a uint32 describing the length of the channel name.
// DataLength is a uint32 describing the length of the data.
type EventHeader struct {
	LCMSyncWord          uint32
	EventNumberUpperBits uint32
	EventNumberLowerBits uint32
	TimestampUpperBits   uint32
	TimestampLowerBits   uint32
	ChannelLength        uint32
	DataLength           uint32
}

// Read reads one LCM event.
func (e *Event) Read(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &e.EventHeader); err != nil {
		return err
	}
	if e.EventHeader.LCMSyncWord != lcmSyncWord {
		return errors.New("not lcm event")
	}
	rawData := make([]byte, e.EventHeader.ChannelLength+e.EventHeader.DataLength)
	n, err := r.Read(rawData)
	if err != nil {
		return nil
	}
	if uint32(n) != e.EventHeader.ChannelLength+e.EventHeader.DataLength {
		return errors.New("incorrect number of bytes read")
	}
	channel, params := split(string(rawData[:e.EventHeader.ChannelLength]), "?")
	e.Channel = channel
	e.Params = params
	e.Data = rawData[e.EventHeader.ChannelLength:]
	p := strings.Split(e.Params, "&")
	if len(p) > 1 || p[0] != "" {
		return xerrors.Errorf("compressed data not supported in log reading")
	}
	return nil
}

// Open returns a LogReader for reading from the specified LCM log.
func Open(s string) (*LogReader, error) {
	f, err := os.Open(s)
	if err != nil {
		return nil, err
	}
	return &LogReader{logFile: f}, nil
}

// ReceiveProto reads a proto LCM message from an LCM log.
func (l *LogReader) ReceiveProto(context.Context) error {
	if err := l.event.Read(l.logFile); err != nil {
		return err
	}
	messageType := proto.MessageType(l.event.Channel)
	if messageType == nil {
		return nil // don't error on encountering non-proto channels
	}
	protoMessage := reflect.New(messageType.Elem()).Interface().(proto.Message)
	if err := proto.Unmarshal(l.event.Data, protoMessage); err != nil {
		return err
	}
	l.protoMessage = protoMessage
	return nil
}

// ProtoMessage returns the last read proto LCM message.
func (l *LogReader) ProtoMessage() proto.Message {
	return l.protoMessage
}

// Close closes the LCM log file.
func (l *LogReader) Close() error {
	return l.logFile.Close()
}
