package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/einride/lcm-go"
	"github.com/einride/lcm-go/pkg/player"
)

func main() {
	cmdAddr := flag.String("address", "localhost:7667", "the address to broadcast messages to")
	cmdDur := flag.Duration("maxtime", time.Hour, "filters out messages with longer time spans")
	cmdSpeedFactor := flag.Float64("speed", 1, "speed factor to play speed")
	flag.Parse()

	if len(flag.Args()) < 1 {
		log.Println("too few arguments")
		return
	}
	fileName := flag.Arg(len(flag.Args()) - 1)
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	udpAddr, err := net.ResolveUDPAddr("udp", *cmdAddr)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	transmitter, err := lcm.DialMulticastUDP(ctx, lcm.WithTransmitAddress(udpAddr), lcm.WithTransmitTTL(1))
	if err != nil {
		log.Fatal(err)
	}

	logPlayer := player.NewPlayer(f, *cmdDur, *cmdSpeedFactor, transmitter)
	length, noMessages, err := logPlayer.GetLength()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	log.Printf("log length is: %s", length)

	skippedMessages, err := logPlayer.Play(ctx, func(messageNumber int) {
		fmt.Printf("\r%s", strings.Repeat(" ", 35))
		fmt.Printf("\rDone... %d / %d packets", messageNumber, noMessages)
	})
	if err != nil {
		log.Fatal(err)
	}
	println("\nfinished, skipped messages", skippedMessages)
}
