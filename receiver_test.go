package lcm

import (
	"context"
	"errors"
	"testing"
	"time"

	mock_lcm "github.com/einride/lcm-go/test/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
)

func TestReceiver_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := mock_lcm.NewMockUDPReader(ctrl)
	rx := NewReceiver(r)
	err := errors.New("foo")
	r.EXPECT().Close().Return(err)
	require.Equal(t, err, rx.Close())
}

func TestReceiver_Receive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := mock_lcm.NewMockUDPReader(ctrl)
	rx := NewReceiver(r)
	data := []byte{
		0x4c, 0x43, 0x30, 0x32, // short header magic
		0x12, 0x34, 0x56, 0x78, // sequence number
		'a', 'b', 'c', 0x00, // channel
		0x01, 0x02, 0x03, // payload
	}
	r.EXPECT().ReadFromUDP(gomock.Any()).Do(func(b []byte) {
		copy(b, data)
	}).Return(len(data), nil, nil)
	deadline := time.Unix(0, 1)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	r.EXPECT().SetReadDeadline(deadline)
	require.NoError(t, rx.Receive(ctx))
	expected := &Message{
		SequenceNumber: 0x12345678,
		Channel:        "abc",
		Data:           []byte{0x01, 0x02, 0x03},
	}
	require.Equal(t, expected, rx.Message())
}

func TestReceiver_Receive_ReadError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := mock_lcm.NewMockUDPReader(ctrl)
	rx := NewReceiver(r)
	err := errors.New("foo")
	r.EXPECT().SetReadDeadline(time.Time{})
	r.EXPECT().ReadFromUDP(gomock.Any()).Return(0, nil, err)
	require.True(t, xerrors.Is(rx.Receive(context.Background()), err))
}

func TestReceiver_Receive_DeadlineError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := mock_lcm.NewMockUDPReader(ctrl)
	rx := NewReceiver(r)
	err := errors.New("foo")
	deadline := time.Unix(0, 1)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	r.EXPECT().SetReadDeadline(deadline).Return(err)
	require.True(t, xerrors.Is(rx.Receive(ctx), err))
}

func TestReceiver_Receive_BadMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := mock_lcm.NewMockUDPReader(ctrl)
	rx := NewReceiver(r)
	data := []byte{
		0x4c, 0x43, 0x30, 0x32, // short header magic
		0x12, 0x34, 0x56,
	}
	r.EXPECT().SetReadDeadline(time.Time{})
	r.EXPECT().ReadFromUDP(gomock.Any()).Do(func(b []byte) {
		copy(b, data)
	}).Return(len(data), nil, nil)
	require.Error(t, rx.Receive(context.Background()))
}
