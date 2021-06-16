package lcmlog

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestMessage_Marshaling(t *testing.T) {
	// Marshalling and Unmarshalling "params" isn't supported. Why?
	originalMsg := Message{
		EventNumber: 1,
		Timestamp:   time.Unix(300, 10e6),
		Channel:     "test",
		Data:        []byte("test_data"),
	}

	var actualMsg Message
	actualMsg.UnmarshalBinary(originalMsg.MarshalBinary())

	assert.DeepEqual(t, originalMsg, actualMsg)
}
