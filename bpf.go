package lcm

import (
	"encoding/binary"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/bpf"
)

// indexOfUDPPayload is the first byte index of the payload in a a UDP packet.
const indexOfUDPPayload = 8

// ShortMessageFilter accepts only LCM short messages.
func ShortMessageFilter() []bpf.Instruction {
	return []bpf.Instruction{
		bpf.LoadAbsolute{Off: indexOfUDPPayload + indexOfHeaderMagic, Size: 4},
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: shortMessageMagic, SkipTrue: 1},
		bpf.RetConstant{Val: lengthOfLargestUDPMessage},
		bpf.RetConstant{Val: 0},
	}
}

// ShortMessageChannelFilter accepts LCM short messages where the channel equals any of the specified channels.
func ShortMessageChannelFilter(channels ...string) []bpf.Instruction {
	const (
		jumpNextChannelPlaceholder = 254
		jumpRejectPlaceholder = 253
		estimatedInstructionsPerChannel = 30
	)
	program := make([]bpf.Instruction, 0, estimatedInstructionsPerChannel*len(channels))
	// accept only short messages
	program = append(program,
		bpf.LoadAbsolute{Off: indexOfUDPPayload + indexOfHeaderMagic, Size: 4},
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: shortMessageMagic, SkipTrue: jumpRejectPlaceholder},
	)
	// check for each channel, accept if any matches
	for _, channel := range channels {
		channel := []byte(channel)
		for i, size := 0, 0; i < len(channel); i += size {
			remaining := channel[i:]
			var val uint32
			switch len(remaining) {
			case 1:
				val = uint32(remaining[0])
				size = 1
			case 2, 3:
				val = uint32(binary.BigEndian.Uint16(remaining))
				size = 2
			default:
				val = binary.BigEndian.Uint32(remaining)
				size = 4
			}
			currByteIndex := indexOfUDPPayload + indexOfChannel + uint32(i)
			program = append(program,
				bpf.LoadAbsolute{Off: currByteIndex, Size: size},
				bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: val, SkipTrue: jumpNextChannelPlaceholder},
			)
		}
		byteIndex := indexOfUDPPayload + indexOfChannel + uint32(len(channel))
		program = append(program,
			bpf.LoadAbsolute{Off: byteIndex, Size: 1},
			// If there is a query parameter accept the message as is.
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: 0, SkipTrue: 1},
			bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: '?', SkipTrue: jumpNextChannelPlaceholder},
			// Channel match, accept package.
			bpf.RetConstant{Val: lengthOfLargestUDPMessage},
		)
	}
	// No channel match, reject package.
	program = append(program, bpf.RetConstant{Val: 0})
	// Start with next channel pointing to the rejection we just added
	nextChannelPos := uint8(len(program)) - 1
	rejectPos := nextChannelPos
	// Now we back-track through the program to find the placeholders and
	// rewrite those to a real jump to the next channel (or the reject instruction).
	for i := uint8(len(program)) - 1; i > 0; i-- {
		switch instr := program[i].(type) {
		case bpf.JumpIf:
			switch instr.SkipTrue {
			case jumpNextChannelPlaceholder :
				instr.SkipTrue = nextChannelPos - i - 1 // Remove one, since the skip is "off by one".
			case jumpRejectPlaceholder :
				instr.SkipTrue = rejectPos - i - 1
			default:
				continue
			}
			program[i] = instr
		case bpf.LoadAbsolute:
			// Each channel matching starts with a load of the first byte, uint16 or uint32 in channel.
			// If it's not, it's not the start of a the channel name
			if instr.Off != indexOfUDPPayload+indexOfChannel {
				continue
			}
			nextChannelPos = i
		}
	}
	return program
}

// ShortProtoMessageFilter accepts LCM short messages where the channel equals any of the proto message names.
func ShortProtoMessageFilter(msgs ...proto.Message) []bpf.Instruction {
	channels := make([]string, len(msgs))
	for i, msg := range msgs {
		channels[i] = proto.MessageName(msg)
	}
	return ShortMessageChannelFilter(channels...)
}
