package lcm_test

import (
	"context"
	"log"
	"net"
	"time"

	"go.einride.tech/lcm"
	"go.einride.tech/lcm/compression/lcmlz4"
	"golang.org/x/net/nettest"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ExampleReceiver() {
	ctx := context.Background()
	ifi, err := nettest.RoutedInterface("ip4", net.FlagUp|net.FlagMulticast)
	if err != nil {
		panic(err) // TODO: Handle error.
	}
	rx, err := lcm.ListenMulticastUDP(
		ctx,
		lcm.WithReceiveInterface(ifi.Name),
		lcm.WithReceiveProtos(&timestamppb.Timestamp{}),
	)
	if err != nil {
		panic(err) // TODO: Handle error.
	}
	defer func() {
		if err := rx.Close(); err != nil {
			panic(err) // TODO: Handle error.
		}
	}()
	log.Printf("listening on: %s\n", ifi.Name)
	for {
		if err := rx.ReceiveProto(ctx); err != nil {
			panic(err) // TODO: Handle error.
		}
		log.Printf("received: %v", rx.ProtoMessage())
	}
}

func ExampleTransmitter() {
	ctx := context.Background()
	ifi, err := nettest.RoutedInterface("ip4", net.FlagUp|net.FlagMulticast)
	if err != nil {
		panic(err) // TODO: Handle error.
	}
	tx, err := lcm.DialMulticastUDP(
		ctx,
		lcm.WithTransmitInterface(ifi.Name),
		lcm.WithTransmitCompressionProto(lcmlz4.NewCompressor(), &timestamppb.Timestamp{}),
	)
	if err != nil {
		panic(err) // TODO: Handle error.
	}
	defer func() {
		if err := tx.Close(); err != nil {
			panic(err) // TODO: Handle error.
		}
	}()
	ticker := time.NewTicker(1000 * time.Millisecond)
	log.Printf("transmitting on: %s\n", ifi.Name)
	for range ticker.C {
		if err := tx.TransmitProto(ctx, timestamppb.Now()); err != nil {
			panic(err) // TODO: Handle error.
		}
	}
}
