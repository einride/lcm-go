package lcm

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	mock_lcm "github.com/einride/lcm-go/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
)

func TestTransmitter_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	w := mock_lcm.NewMockUDPWriter(ctrl)
	tx := NewTransmitter(w)
	err := errors.New("foo")
	w.EXPECT().Close().Return(err)
	require.Equal(t, err, tx.Close())
}

func TestTransmitter_Transmit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	w := mock_lcm.NewMockUDPWriter(ctrl)
	tx := NewTransmitter(w)
	deadline := time.Unix(0, 1)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	expected := []byte{
		0x4c, 0x43, 0x30, 0x32, // short header magic
		0x00, 0x00, 0x00, 0x00, // sequence number
		'f', 'o', 'o', 0x00, // channel
		0x01, 0x02, 0x03, // payload
	}
	w.EXPECT().SetWriteDeadline(deadline)
	w.EXPECT().Write(gomock.Any()).Do(func(data []byte) {
		require.Equal(t, expected, data)
	})
	require.NoError(t, tx.Transmit(ctx, "foo", []byte{0x01, 0x02, 0x03}))
}

func TestTransmitter_TransmitProto(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	w := mock_lcm.NewMockUDPWriter(ctrl)
	tx := NewTransmitter(w)
	deadline := time.Unix(0, 1)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	protoMsg := &timestamp.Timestamp{Seconds: 1, Nanos: 2}
	expected := []byte{
		0x4c, 0x43, 0x30, 0x32, // short header magic
		0x00, 0x00, 0x00, 0x00, // sequence number
		'f', 'o', 'o', 0x00, // channel
		0x08, 0x01, 0x10, 0x02, // protobuf message payload
	}
	w.EXPECT().SetWriteDeadline(deadline)
	w.EXPECT().Write(gomock.Any()).Do(func(data []byte) {
		require.Equal(t, expected, data)
	})
	require.NoError(t, tx.TransmitProto(ctx, "foo", protoMsg))
}

func TestTransmitter_Transmit_WriteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	w := mock_lcm.NewMockUDPWriter(ctrl)
	tx := NewTransmitter(w)
	err := errors.New("boom")
	w.EXPECT().SetWriteDeadline(time.Time{})
	w.EXPECT().Write(gomock.Any()).Return(0, err)
	require.True(t, xerrors.Is(tx.Transmit(context.Background(), "foo", []byte{}), err))
}

func TestTransmitter_DeadlineError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	w := mock_lcm.NewMockUDPWriter(ctrl)
	tx := NewTransmitter(w)
	deadline := time.Unix(0, 1)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	err := errors.New("boom")
	w.EXPECT().SetWriteDeadline(deadline).Return(err)
	require.True(t, xerrors.Is(tx.Transmit(ctx, "foo", []byte{}), err))
}

func TestTransmitter_Transmit_BadChannel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	w := mock_lcm.NewMockUDPWriter(ctrl)
	tx := NewTransmitter(w)
	badChannel := strings.Repeat("a", lengthOfLongestChannel+1)
	require.Error(t, tx.Transmit(context.Background(), badChannel, []byte{}))
}

func TestTransmitter_Transmit_BadData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	w := mock_lcm.NewMockUDPWriter(ctrl)
	tx := NewTransmitter(w)
	badData := make([]byte, lengthOfLargestPayload+1)
	require.Error(t, tx.Transmit(context.Background(), "foo", badData))
}
