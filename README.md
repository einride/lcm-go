# LCM Go

Performant, native Go [LCM](https://lcm-proj.github.io/) implementation
with integrated support for [protobuf][protobuf] and compression.

[protobuf]: https://developers.google.com/protocol-buffers

## Installation

`go get go.einride.tech/lcm`

## Usage

### Receiver

```go
    rx, err := lcm.ListenMulticastUDP(
     	ctx,
     	lcm.WithReceiveInterface("eth0"),
     	lcm.WithReceiveProtos(&timestamppb.Timestamp{}),
    )
    if err != nil {
        panic(err) // TODO: Handle error.
    }
    if err := rx.ReceiveProto(ctx); err != nil {
        panic(err) // TODO: Handle error.
    }
    log.Println(rx.ProtoMessage())
```

### Transmitter

```go
    tx, err := lcm.DialMulticastUDP(
        ctx,
        lcm.WithTransmitInterface("eth0"),
    )
    if err != nil {
        panic(err) // TODO: Handle error.
    }
    if err := tx.TransmitProto(ctx, &timestamppb.Now()); err != nil {
        panic(err) // TODO: Handle error.
    }
```

## Notable features

### Protobuf messages

Protobuf messages can be transmitted and received, with encoding and decoding
handled by the LCM stack.

### Compression

The library can handle compression and decompression of messages at the
channel-level, with the compression scheme indicated by a query-parameter on the
channel name (similar to HTTP URLs).

For example an LZ4 compressed message transmitted over a channel named
`google.protobuf.Timestamp?z=lz4` will be automatically decompressed.

### BPF filtering

When specifying a set of channels to receive from, the library will attempt to
use BPF filters to only receive messages from those channels from the kernel.

However, since there is a limit of 255 instructions on BPF filters, if there are
too many channels, it will fallback and listening to everything.

## Notable missing features

### Fragmented messages

This library currently does not support fragmented messages.
