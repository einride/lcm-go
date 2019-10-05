package lcm

import (
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/bpf"
)

// indexOfUDPPayload is the first byte index of the payload in a a UDP packet.
const (
	indexOfUDPPayload          = 8
	jumpNextChannelPlaceholder = 254
)

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
	const estimatedInstructionsPerChannel = 30
	program := make([]bpf.Instruction, 0, estimatedInstructionsPerChannel*len(channels))
	// accept only short messages
	program = append(program,
		bpf.LoadAbsolute{Off: indexOfUDPPayload + indexOfHeaderMagic, Size: 4},
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: shortMessageMagic, SkipTrue: 1},
		bpf.RetConstant{Val: 0},
	)
	// check for each channel, accept if any matches
	for _, channel := range channels {
		for i := 0; i < len(channel); i++ {
			// check if the i:th byte matches, skip to next channel if not
			currByteIndex := indexOfUDPPayload + indexOfChannel + uint32(i)
			program = append(program,
				bpf.LoadAbsolute{Off: currByteIndex, Size: 1},
				bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: uint32(channel[i]), SkipTrue: jumpNextChannelPlaceholder},
			)
		}
		byteIndex := indexOfUDPPayload + indexOfChannel + uint32(len(channel))
		program = append(program,
			bpf.LoadAbsolute{Off: byteIndex, Size: 1},
			// If there is a query parameter accept the message as is
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: '?', SkipTrue: 1},
			bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 0, SkipTrue: 1},
			// channel match, accept package
			bpf.RetConstant{Val: lengthOfLargestUDPMessage},
		)
	}
	// no channel match, reject package
	program = append(program, bpf.RetConstant{Val: 0})
	// fill in the missing instruction
	nextChannelPos := uint8(len(program)) - 1
	for i := uint8(len(program)) - 1; i >= 3; i-- {
		switch instr := program[i].(type) {
		case bpf.JumpIf:
			if instr.SkipTrue != jumpNextChannelPlaceholder {
				continue
			}
			instr.SkipTrue = nextChannelPos - i - 1 // remove one, since the skip is "off by one"
			program[i] = instr
		case bpf.LoadAbsolute:
			// Each channel starts with a load of the first byte in channel
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
	channels := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		channels = append(channels, proto.MessageName(msg))
	}
	return ShortMessageChannelFilter(channels...)
}
