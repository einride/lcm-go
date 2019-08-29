package lcmlog

import (
	"encoding/binary"
	"encoding/hex"
	"time"

	"golang.org/x/xerrors"
)

// from: https://lcm-proj.github.io/log_file_format.html
//
// Event Encoding
//
// Each event is encoded as a binary structure consisting of a header, followed by the channel and the data.
//
// The header is 28 bytes and has the following format:
//
//  0      7 8     15 16    23 24    31
//  +--------+--------+--------+--------+
//  |   LCM Sync Word                   |
//  +--------+--------+--------+--------+
//  |   Event Number (upper 32 bits)    |
//  +--------+--------+--------+--------+
//  |   Event Number (lower 32 bits)    |
//  +--------+--------+--------+--------+
//  |   Timestamp (upper 32 bits)       |
//  +--------+--------+--------+--------+
//  |   Timestamp (lower 32 bits)       |
//  +--------+--------+--------+--------+
//  |   Channel Length                  |
//  +--------+--------+--------+--------+
//  |   Data Length                     |
//  +--------+--------+--------+--------+
//
// LCM Sync Word is an unsigned 32-bit integer with value 0xEDA1DA01
//
// Event Number and Timestamp fields of the header are as described above.
//
// Channel Length is an unsigned 32-bit integer describing the length of the channel name.
//
// Data Length is an unsigned 32-bit integer describing the length of the data.
//
// Each header is immediately followed by the UTF-8 encoding of the LCM channel, and then the message data.
// The channel is not NULL-terminated.
//
// All integers are packed in network order (big endian)
const (
	syncWord              = 0xeda1da01
	lengthOfSyncWord      = 4
	lengthOfEventNumber   = 8
	lengthOfTimestamp     = 8
	lengthOfChannelLength = 4
	lengthOfDataLength    = 4
	lengthOfHeader        = lengthOfSyncWord +
		lengthOfEventNumber +
		lengthOfTimestamp +
		lengthOfChannelLength +
		lengthOfDataLength
	indexOfSyncWord      = 0
	endOfSyncWord        = indexOfSyncWord + lengthOfSyncWord
	indexOfEventNumber   = endOfSyncWord
	endOfEventNumber     = indexOfEventNumber + lengthOfEventNumber
	indexOfTimestamp     = endOfEventNumber
	endOfTimestamp       = indexOfTimestamp + lengthOfTimestamp
	indexOfChannelLength = endOfTimestamp
	endOfChannelLength   = indexOfChannelLength + lengthOfChannelLength
	indexOfDataLength    = endOfChannelLength
	endOfDataLength      = indexOfDataLength + lengthOfDataLength
	indexOfChannel       = endOfDataLength
)

type Message struct {
	EventNumber uint64
	Timestamp   time.Time
	Channel     []byte
	Data        []byte
}

func (m *Message) unmarshalBinary(b []byte) {
	m.EventNumber = binary.BigEndian.Uint64(b[indexOfEventNumber:endOfEventNumber])
	timestampMicros := binary.BigEndian.Uint64(b[indexOfTimestamp:endOfTimestamp])
	m.Timestamp = time.Unix(0, (time.Duration(timestampMicros) * time.Microsecond).Nanoseconds())
	channelLength := binary.BigEndian.Uint32(b[indexOfChannelLength:endOfChannelLength])
	dataLength := binary.BigEndian.Uint32(b[indexOfDataLength:endOfDataLength])
	endOfChannel := indexOfChannel + channelLength
	indexOfData := endOfChannel
	endOfData := indexOfData + dataLength
	m.Channel = b[indexOfChannel:endOfChannel]
	m.Data = b[indexOfData:endOfData]
}

func scanLogMessages(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if len(data) == 0 {
		return 0, nil, nil
	}
	if len(data) < lengthOfHeader {
		if atEOF {
			return 0, nil, xerrors.Errorf("partial message at end of log file: %s", hex.EncodeToString(data))
		}
		return 0, nil, nil
	}
	actualSyncWord := binary.BigEndian.Uint32(data[indexOfSyncWord:endOfSyncWord])
	if actualSyncWord != syncWord {
		return 0, nil, xerrors.Errorf("unexpected sync word: %#x", actualSyncWord)
	}
	channelLength := binary.BigEndian.Uint32(data[indexOfChannelLength:endOfChannelLength])
	dataLength := binary.BigEndian.Uint32(data[indexOfDataLength:endOfDataLength])
	messageLength := lengthOfHeader + int(channelLength) + int(dataLength)
	if len(data) < messageLength {
		if atEOF {
			return 0, nil, xerrors.Errorf("partial message at end of log file: %s", hex.EncodeToString(data))
		}
		return 0, nil, nil
	}
	return messageLength, data[:messageLength], nil
}
