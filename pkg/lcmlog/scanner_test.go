package lcmlog

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"
)

var _ = io.WriterTo(&Message{})

func TestScanner_Scan_Testdata(t *testing.T) {
	f, err := os.Open("testdata/lcmlog.00")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
	}()
	sc := NewScanner(f)
	var i int32
	for sc.Scan() {
		require.Equal(t, []byte("test"), sc.Message().Channel)
		require.Equal(t, uint64(i), sc.Message().EventNumber)
		ts := &timestamp.Timestamp{}
		require.NoError(t, proto.Unmarshal(sc.Message().Data, ts))
		require.Equal(t, &timestamp.Timestamp{Nanos: i}, ts)
		i++
	}
}

func TestMarshalling(t *testing.T) {
	m := Message{EventNumber: 1, Timestamp: time.Unix(300, 10e6), Channel: []byte("test"), Data: []byte("test_data")}
	var newM Message
	newM.unmarshalBinary(m.marshalBinary())
	require.Equal(t, m.Timestamp, newM.Timestamp)
	require.Equal(t, m, newM)
}

func TestWriteToFile(t *testing.T) {
	fileName := "test.log"
	file, err := os.Create(fileName)
	require.NoError(t, err)
	messages := []*Message{
		{EventNumber: 1, Timestamp: time.Unix(300, 10e6), Channel: []byte("test"), Data: []byte("test_data")},
		{EventNumber: 8, Timestamp: time.Unix(300, 10e6), Channel: []byte("testt"), Data: []byte("test_data2")},
	}
	messagesMap := map[uint64]*Message{}
	for i := range messages {
		m := *messages[i]
		_, ok := messagesMap[m.EventNumber]
		require.False(t, ok)
		messagesMap[m.EventNumber] = &m
	}
	for i := range messages {
		_, err = messages[i].WriteTo(file)
		require.NoError(t, err)
	}
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)
	newfile, err := os.Open(fileName)
	require.NoError(t, err)
	scanner := NewScanner(newfile)
	newMessages := []*Message{}
	for scanner.Scan() {
		m := scanner.Message()
		require.Equal(t, messagesMap[m.EventNumber], m)
		newMessages = append(newMessages, m)
	}
	require.Equal(t, len(messages), len(newMessages))
	err = os.Remove(fileName)
	require.NoError(t, err)
}

func TestWriteToFileFromFile(t *testing.T) {
	f, err := os.Open("testdata/lcmlog.00")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
	}()
	sc := NewScanner(f)
	messages := []*Message{}
	messagesMap := map[uint64]*Message{}
	for sc.Scan() {
		m := *sc.Message()
		messages = append(messages, &m)
		messagesMap[sc.Message().EventNumber] = &m
	}
	fileName := "test.log"
	file, err := os.Create(fileName)
	require.NoError(t, err)
	for i := range messages {
		_, err = messages[i].WriteTo(file)
		require.NoError(t, err)
	}
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)
	newfile, err := os.Open(fileName)
	require.NoError(t, err)
	scanner := NewScanner(newfile)
	newMessages := []*Message{}
	for scanner.Scan() {
		m := messagesMap[scanner.Message().EventNumber]
		newMessage := *scanner.Message()
		require.Equal(t, m, &newMessage)
		newMessages = append(newMessages, &newMessage)
	}
	require.Equal(t, len(messages), len(newMessages))
	require.Equal(t, messages, newMessages)
	err = os.Remove(fileName)
	require.NoError(t, err)
}
