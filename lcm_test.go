package lcm

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/einride/lcm-go/pkg/lz4"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
	"golang.org/x/net/nettest"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/testing/protocmp"
	"gotest.tools/v3/assert"
)

func TestLCM_OneTransmitter_OneReceiver(t *testing.T) {
	// setup
	const testTimeout = 1 * time.Second
	ip := net.IPv4(239, 0, 0, 1)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	freePort := getFreePort(t)
	ifi := getInterface(t)
	rx, err := ListenMulticastUDP(
		ctx,
		WithReceiveInterface(ifi.Name),
		WithReceivePort(freePort),
		WithReceiveAddress(ip),
	)
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, rx.Close())
	}()
	tx, err := DialMulticastUDP(
		ctx,
		WithTransmitInterface(ifi.Name),
		WithTransmitAddress(&net.UDPAddr{IP: ip, Port: freePort}),
		WithTransmitCompression(lz4.NewCompressor(), "first"),
	)
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, tx.Close())
	}()
	t.Run("receive first", func(t *testing.T) {
		// when the receiver receives
		var g errgroup.Group
		g.Go(func() error {
			return rx.Receive(ctx)
		})
		// and the transmitter transmits
		assert.NilError(t, tx.Transmit(ctx, "first", []byte("foo")))
		// then the receiver should receive the transmitted message
		assert.NilError(t, g.Wait())
		assert.Equal(t, "first", rx.Message().Channel)
		assert.DeepEqual(t, []byte("foo"), rx.Message().Data)
		assert.Equal(t, uint32(0), rx.Message().SequenceNumber)
	})
	t.Run("receive second", func(t *testing.T) {
		// when the receiver receives
		var g errgroup.Group
		g.Go(func() error {
			return rx.Receive(ctx)
		})
		// and the transmitter transmits
		assert.NilError(t, tx.Transmit(ctx, "second", []byte("bar")))
		// then the receiver should receive the transmitted message
		assert.NilError(t, g.Wait())
		assert.Equal(t, "second", rx.Message().Channel)
		assert.DeepEqual(t, []byte("bar"), rx.Message().Data)
		assert.Equal(t, uint32(1), rx.Message().SequenceNumber)
	})
}

func TestLCM_OneTransmitter_MultipleReceivers(t *testing.T) {
	// setup
	const testTimeout = 1 * time.Second
	ip1 := net.IPv4(239, 0, 0, 1)
	ip2 := net.IPv4(239, 0, 0, 2)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	freePort := getFreePort(t)
	ifi := getInterface(t)
	rx1, err := ListenMulticastUDP(
		ctx,
		WithReceiveInterface(ifi.Name),
		WithReceivePort(freePort),
		WithReceiveAddress(ip1),
	)
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, rx1.Close())
	}()
	rx2, err := ListenMulticastUDP(
		ctx,
		WithReceiveInterface(ifi.Name),
		WithReceivePort(freePort),
		WithReceiveAddress(ip2),
	)
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, rx2.Close())
	}()
	tx, err := DialMulticastUDP(
		ctx,
		WithTransmitInterface(ifi.Name),
		WithTransmitAddress(&net.UDPAddr{IP: ip1, Port: freePort}),
		WithTransmitAddress(&net.UDPAddr{IP: ip2, Port: freePort}),
	)
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, tx.Close())
	}()
	t.Run("receive", func(t *testing.T) {
		// when the receiver receives
		var g errgroup.Group
		g.Go(func() error {
			return rx1.Receive(ctx)
		})
		g.Go(func() error {
			return rx2.Receive(ctx)
		})
		// and the transmitter transmits
		assert.NilError(t, tx.Transmit(ctx, "foo", []byte("bar")))
		// then the receiver should receive the transmitted message
		assert.NilError(t, g.Wait())
		for _, rx := range []*Receiver{rx1, rx2} {
			assert.Equal(t, "foo", rx.Message().Channel)
			assert.DeepEqual(t, []byte("bar"), rx.Message().Data)
			assert.Equal(t, uint32(0), rx.Message().SequenceNumber)
		}
	})
}

func TestLCM_OneReceiver_MultipleTransmitters(t *testing.T) {
	// setup
	const testTimeout = 1 * time.Second
	ip1 := net.IPv4(239, 0, 0, 1)
	ip2 := net.IPv4(239, 0, 0, 2)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	freePort := getFreePort(t)
	ifi := getInterface(t)
	rx, err := ListenMulticastUDP(
		ctx,
		WithReceiveInterface(ifi.Name),
		WithReceivePort(freePort),
		WithReceiveAddress(ip1),
		WithReceiveAddress(ip2),
	)
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, rx.Close())
	}()
	tx1, err := DialMulticastUDP(
		ctx,
		WithTransmitInterface(ifi.Name),
		WithTransmitAddress(&net.UDPAddr{IP: ip1, Port: freePort}),
	)
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, tx1.Close())
	}()
	tx2, err := DialMulticastUDP(
		ctx,
		WithTransmitInterface(ifi.Name),
		WithTransmitAddress(&net.UDPAddr{IP: ip2, Port: freePort}),
	)
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, tx2.Close())
	}()
	t.Run("receive first", func(t *testing.T) {
		// when the receiver receives
		var g errgroup.Group
		g.Go(func() error {
			return rx.Receive(ctx)
		})
		// and the transmitter transmits
		assert.NilError(t, tx1.Transmit(ctx, "1", []byte("data1")))
		// then the receiver should receive the transmitted message
		assert.NilError(t, g.Wait())
		assert.Equal(t, "1", rx.Message().Channel)
		assert.DeepEqual(t, []byte("data1"), rx.Message().Data)
		assert.Equal(t, uint32(0), rx.Message().SequenceNumber)
	})
	t.Run("receive second", func(t *testing.T) {
		// when the receiver receives
		var g errgroup.Group
		g.Go(func() error {
			return rx.Receive(ctx)
		})
		// and the transmitter transmits
		assert.NilError(t, tx2.Transmit(ctx, "2", []byte("data2")))
		// then the receiver should receive the transmitted message
		assert.NilError(t, g.Wait())
		assert.Equal(t, "2", rx.Message().Channel)
		assert.DeepEqual(t, []byte("data2"), rx.Message().Data)
		assert.Equal(t, uint32(0), rx.Message().SequenceNumber)
	})
}

func TestLCM_OneTransmitter_OneReceiver_ManyCompressed(t *testing.T) {
	// setup
	const testTimeout = 1 * time.Second
	ip := net.IPv4(239, 0, 0, 1)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	freePort := getFreePort(t)
	ifi := getInterface(t)
	rx, err := ListenMulticastUDP(
		ctx,
		WithReceiveInterface(ifi.Name),
		WithReceivePort(freePort),
		WithReceiveAddress(ip),
	)
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, rx.Close())
	}()
	tx, err := DialMulticastUDP(
		ctx,
		WithTransmitInterface(ifi.Name),
		WithTransmitAddress(&net.UDPAddr{IP: ip, Port: freePort}),
		WithTransmitCompression(lz4.NewCompressor(), "first"),
	)
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, tx.Close())
	}()
	for i := 100; i < 110; i++ {
		i := i
		t.Run("receive first", func(t *testing.T) {
			// when the receiver receives
			var g errgroup.Group
			g.Go(func() error {
				return rx.Receive(ctx)
			})
			// and the transmitter transmits
			assert.NilError(t, tx.Transmit(ctx, "first", []byte(strings.Repeat("foo", i))))
			// then the receiver should receive the transmitted message
			assert.NilError(t, g.Wait())
			assert.Equal(t, "first", rx.Message().Channel)
			assert.DeepEqual(t, []byte(strings.Repeat("foo", i)), rx.Message().Data)
			assert.Equal(t, uint32(i-100), rx.Message().SequenceNumber)
		})
	}
}

func TestLCM_ProtoTransmitter_ProtoReceiver(t *testing.T) {
	// setup
	const testTimeout = 1 * time.Second
	ip := net.IPv4(239, 0, 0, 1)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	freePort := getFreePort(t)
	ifi := getInterface(t)
	rx, err := ListenMulticastUDP(
		ctx,
		WithReceiveInterface(ifi.Name),
		WithReceivePort(freePort),
		WithReceiveAddress(ip),
		WithReceiveProtos(
			&timestamp.Timestamp{},
			&duration.Duration{},
		),
	)
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, rx.Close())
	}()
	tx, err := DialMulticastUDP(
		ctx,
		WithTransmitInterface(ifi.Name),
		WithTransmitAddress(&net.UDPAddr{IP: ip, Port: freePort}),
		WithTransmitCompressionProto(lz4.NewCompressor(), &timestamp.Timestamp{}, &duration.Duration{}),
	)
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, tx.Close())
	}()
	t.Run("receive first", func(t *testing.T) {
		// when the receiver receives
		var g errgroup.Group
		g.Go(func() error {
			return rx.ReceiveProto(ctx)
		})
		// and the transmitter transmits
		assert.NilError(t, tx.TransmitProto(ctx, &timestamp.Timestamp{Seconds: 1, Nanos: 2}))
		// then the receiver should receive the transmitted message
		assert.NilError(t, g.Wait())
		assert.Equal(t, "google.protobuf.Timestamp", rx.Message().Channel)
		assert.DeepEqual(t, &timestamp.Timestamp{Seconds: 1, Nanos: 2}, rx.ProtoMessage(), protocmp.Transform())
	})
	t.Run("receive second", func(t *testing.T) {
		// when the receiver receives
		var g errgroup.Group
		g.Go(func() error {
			return rx.ReceiveProto(ctx)
		})
		// and the transmitter transmits
		assert.NilError(t, tx.TransmitProto(ctx, &duration.Duration{Seconds: 1, Nanos: 2}))
		// then the receiver should receive the transmitted message
		assert.NilError(t, g.Wait())
		assert.Equal(t, "google.protobuf.Duration", rx.Message().Channel)
		assert.DeepEqual(t, &duration.Duration{Seconds: 1, Nanos: 2}, rx.ProtoMessage(), protocmp.Transform())
	})
}

func getInterface(t *testing.T) *net.Interface {
	t.Helper()
	ifi, err := nettest.RoutedInterface("ip4", net.FlagUp|net.FlagMulticast|net.FlagLoopback)
	if err == nil {
		return ifi
	}
	ifi, err = nettest.RoutedInterface("ip4", net.FlagUp|net.FlagMulticast)
	assert.NilError(t, err)
	return ifi
}

func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := nettest.NewLocalPacketListener("udp4")
	assert.NilError(t, err)
	defer func() {
		assert.NilError(t, l.Close())
	}()
	return l.LocalAddr().(*net.UDPAddr).Port
}
