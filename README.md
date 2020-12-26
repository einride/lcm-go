# LCM Go

Performant Native Go [LCM](https://lcm-proj.github.io/) implementation
with integrated support for protobuf and compression.

# Installation

`go get -u go.einride.tech/lcm`

Note Einride LCM-Go only supports short messages

# Abstract

This library is designed around 2 design choices to make it easier to
manage LCM on larger deploys, and thereby operate on a higher level
than the original LCM project libraries does.

1. Send and receive methods operate on Google protobufs and
   let the library do the serialization and deserialization.

2. Reuse the protobuf names as channel names, for zero
   configuration. So a channel name will look something like this:
   `google.protobuf.Timestamp`.

That said, the library still supports lower access too.

# Restrictions

The library only supports short messages. So this library is not
suitable if you have to send messages that are larger than 64k
(compression might help you if you can guarantee that the messages
never get larger than that compressed)

# Features

This library has two extra features embedded, compression and BPF filter.

## Compression

To save bandwidth on the network, and saving space for stored logs,
this library supports compression. The configuration only done on the
serverside and added a query-parameter (similar to HTTP URLs) which
tells the receiver if the channel has compression. Currently, the only
supported compression scheme is LZ4.

Compression can be enabled per message type, as some really short
messages actually become larger from compression. So make sure to
benchmark each message type before turning it on.

A compressed message over the channel will look like this:
`google.protobuf.Timestamp?z=lz4`

This means that you can dynamically switch between compression and
non-compression at runtime, or different senders can run with
compression on or off.

## BPF filters

To reduce the load on the receiver side, we added a BPF filter that
filters out channels that we are not listening to. However, since
there is a limit of 255 instructions on BPF filters, if there are too
many channels, it will fallback and listening to everything.

# Examples

For a fully working example: [receiver](./examples/receiver/main.go)
and [sender](./examples/sender/main.go)

Receiver:

```go
    rx, _ := lcm.ListenMulticastUDP(
     	ctx,
     	lcm.WithReceiveInterface("eth0"),
     	lcm.WithReceiveProtos(&timestamppb.Timestamp{}),
    )
    _ = rx.ReceiveProto(ctx)
    switch msg := rx.ProtoMessage().(type) {
    	case *timestamppb.Timestamp:
    		fmt.Println("received time: ", msg.AsTime())
    	}
    }
```

Sender:

```go
    tx, _ := lcm.DialMulticastUDP(
        ctx,
        lcm.WithTransmitInterface("eth0"),
    )
    now := time.Now()
    tx.TransmitProto(ctx, &timestamppb.Timestamp{Seconds: now.Unix(), Nanos: int32(now.Nanosecond())})
```
