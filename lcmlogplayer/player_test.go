package lcmlogplayer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mocklcmlogplayer "go.einride.tech/lcm/test/mocks/lcmlogplayer"
	"gotest.tools/v3/assert"
)

func TestGetLength(t *testing.T) {
	// given
	ctrl := gomock.NewController(t)
	dur, _ := time.ParseDuration("1m39.053644s")
	f, _ := os.Open("testdata/lcmlog.00")
	transmitter := mocklcmlogplayer.NewMockTransmitter(ctrl)
	player := New(f, time.Second, 1, transmitter)
	// then
	outputDuration, messages, err := player.GetLength()
	assert.NilError(t, err)
	// then
	assert.Equal(t, messages, 100)
	assert.Equal(t, dur, outputDuration)
}

func TestPlay(t *testing.T) {
	// given
	ctrl := gomock.NewController(t)
	f, _ := os.Open("testdata/lcmlog.00")
	transmitter := mocklcmlogplayer.NewMockTransmitter(ctrl)
	player := New(f, time.Second, 1, transmitter)

	// then
	skippedMsgs, err := player.Play(context.Background(), func(messageNumber int) {
	})
	assert.NilError(t, err)
	// then
	assert.Equal(t, skippedMsgs, 99)
}
