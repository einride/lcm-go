package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"go.einride.tech/lcm"
	"go.einride.tech/lcm/compression/lcmlz4"
	"golang.org/x/net/nettest"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	ctx := context.Background()
	ifi, err := nettest.RoutedInterface("ip4", net.FlagUp|net.FlagMulticast)
	if err != nil {
		log.Fatalf("failed to get interface: %v", err)
	}
	tx, err := lcm.DialMulticastUDP(
		ctx,
		lcm.WithTransmitInterface(ifi.Name),
		lcm.WithTransmitCompressionProto(lcmlz4.NewCompressor(), &timestamppb.Timestamp{}),
	)
	if err != nil {
		log.Fatalf("failed to start: %v", err)
	}
	defer tx.Close()
	ticker := time.NewTicker(1000 * time.Millisecond)
	fmt.Printf("transmitting on: %s\n", ifi.Name)
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			tx.TransmitProto(ctx, &timestamppb.Timestamp{Seconds: now.Unix(), Nanos: int32(now.Nanosecond())})
		}
	}
}
