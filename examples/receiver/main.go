package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/einride/lcm-go"
	"golang.org/x/net/nettest"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	ctx := context.Background()
	ifi, err := nettest.RoutedInterface("ip4", net.FlagUp|net.FlagMulticast)
	if err != nil {
		log.Fatalf("failed to get interface: %v", err)
	}
	rx, err := lcm.ListenMulticastUDP(
		ctx,
		lcm.WithReceiveInterface(ifi.Name),
		lcm.WithReceiveProtos(&timestamppb.Timestamp{}),
	)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf("listening on: %s\n", ifi.Name)
	for {
		err := rx.ReceiveProto(ctx)
		if err != nil {
			log.Fatalf("failed to receive: %v", err)
			break
		}
		switch msg := rx.ProtoMessage().(type) {
		case *timestamppb.Timestamp:
			fmt.Println("received time: ", msg.AsTime())
		}
	}
}
