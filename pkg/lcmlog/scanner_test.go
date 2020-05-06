package lcmlog

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"gotest.tools/v3/assert"
)

var _ = io.WriterTo(&Message{})

func TestScanner_Scan_Testdata(t *testing.T) {
	f, err := os.Open("testdata/lcmlog.00")
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, f.Close())
	}()
	sc := NewScanner(f)
	var i int32
	for sc.Scan() {
		assert.Equal(t, "test", sc.Message().Channel)
		assert.Equal(t, uint64(i), sc.Message().EventNumber)
		ts := &timestamp.Timestamp{}
		assert.NilError(t, proto.Unmarshal(sc.Message().Data, ts))
		assert.DeepEqual(t, &timestamp.Timestamp{Nanos: i}, ts, protocmp.Transform())
		i++
	}
}

func TestMarshalling(t *testing.T) {
	m := Message{EventNumber: 1, Timestamp: time.Unix(300, 10e6), Channel: "test", Data: []byte("test_data")}
	var newM Message
	newM.unmarshalBinary(m.marshalBinary())
	assert.DeepEqual(t, m.Timestamp, newM.Timestamp)
	assert.DeepEqual(t, m, newM)
}

func TestWriteToFile(t *testing.T) {
	fileName := "test.log"
	file, err := os.Create(fileName)
	assert.NilError(t, err)
	messages := []*Message{
		{EventNumber: 1, Timestamp: time.Unix(300, 10e6), Channel: "test", Data: []byte("test_data")},
		{EventNumber: 8, Timestamp: time.Unix(300, 10e6), Channel: "testt", Data: []byte("test_data2")},
	}
	messagesMap := map[uint64]*Message{}
	for i := range messages {
		m := *messages[i]
		_, ok := messagesMap[m.EventNumber]
		assert.Assert(t, !ok)
		messagesMap[m.EventNumber] = &m
	}
	for i := range messages {
		_, err = messages[i].WriteTo(file)
		assert.NilError(t, err)
	}
	assert.NilError(t, err)
	err = file.Close()
	assert.NilError(t, err)
	newfile, err := os.Open(fileName)
	assert.NilError(t, err)
	scanner := NewScanner(newfile)
	newMessages := []*Message{}
	for scanner.Scan() {
		m := scanner.Message()
		assert.DeepEqual(t, messagesMap[m.EventNumber], m)
		newMessages = append(newMessages, m)
	}
	assert.Equal(t, len(messages), len(newMessages))
	err = os.Remove(fileName)
	assert.NilError(t, err)
}

func TestWriteToFileFromFile(t *testing.T) {
	f, err := os.Open("testdata/lcmlog.00")
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, f.Close())
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
	assert.NilError(t, err)
	for i := range messages {
		_, err = messages[i].WriteTo(file)
		assert.NilError(t, err)
	}
	assert.NilError(t, err)
	err = file.Close()
	assert.NilError(t, err)
	newfile, err := os.Open(fileName)
	assert.NilError(t, err)
	scanner := NewScanner(newfile)
	newMessages := []*Message{}
	for scanner.Scan() {
		m := messagesMap[scanner.Message().EventNumber]
		newMessage := *scanner.Message()
		assert.DeepEqual(t, m, &newMessage)
		newMessages = append(newMessages, &newMessage)
	}
	assert.Equal(t, len(messages), len(newMessages))
	assert.DeepEqual(t, messages, newMessages)
	err = os.Remove(fileName)
	assert.NilError(t, err)
}

func TestScanner_Scan_Compressed_Testdata(t *testing.T) {
	f, err := os.Open("testdata/lcmlog_compressed.00")
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, f.Close())
	}()
	sc := NewScanner(f)
	var j int32
	for i := 100; i < 110; i++ {
		assert.Assert(t, sc.Scan())
		assert.DeepEqual(t, sc.Message().Data, []byte(strings.Repeat("foo", i)))
		assert.Equal(t, "first", sc.Message().Channel)
		assert.Equal(t, uint64(j), sc.Message().EventNumber)
		j++
	}
}
