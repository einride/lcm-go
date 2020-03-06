package lcm

import (
	"encoding/binary"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/bpf"
)

// indexOfUDPPayload is the first byte index of the payload in a a UDP packet.
const (
	indexOfUDPPayload = 8
	offsetHeaderMagic = indexOfUDPPayload + indexOfHeaderMagic
	offsetChannel     = indexOfUDPPayload + indexOfChannel
)

// shortMessageFilter accepts only LCM short messages.
func shortMessageFilter() []bpf.Instruction {
	return []bpf.Instruction{
		bpf.LoadAbsolute{Off: offsetHeaderMagic, Size: 4},
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: shortMessageMagic, SkipTrue: 1},
		bpf.RetConstant{Val: lengthOfLargestUDPMessage},
		bpf.RetConstant{Val: 0},
	}
}

// shortMessageChannelFilter accepts LCM short messages where the channel equals any of the specified channels.
func shortMessageChannelFilter(channels ...string) []bpf.Instruction {
	const (
		jumpNextChannelPlaceholder = 255 - iota
		jumpRejectPlaceholder
		jumpAcceptPlaceholder
		estimatedInstructionsPerChannel = 30
	)
	program := make([]bpf.Instruction, 0, estimatedInstructionsPerChannel*len(channels))
	// accept only short messages
	program = append(program,
		bpf.LoadAbsolute{Off: offsetHeaderMagic, Size: 4},
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
				val, size = uint32(remaining[0]), 1
			case 2, 3:
				val, size = uint32(binary.BigEndian.Uint16(remaining)), 2
			default:
				val, size = binary.BigEndian.Uint32(remaining), 4
			}
			program = append(program,
				bpf.LoadAbsolute{Off: offsetChannel + uint32(i), Size: size},
				bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: val, SkipTrue: jumpNextChannelPlaceholder},
			)
		}
		program = append(program,
			bpf.LoadAbsolute{Off: offsetChannel + uint32(len(channel)), Size: 1},
			// Channel match, accept package.
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: 0, SkipTrue: jumpAcceptPlaceholder},
			// Or if there is a query parameter accept the message as is.
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: '?', SkipTrue: jumpAcceptPlaceholder},
		)
	}
	program = append(program,
		bpf.RetConstant{Val: 0},                         // No channel match, reject package.
		bpf.RetConstant{Val: lengthOfLargestUDPMessage}, // Accept instruction
	)
	// Start with next channel pointing to the rejection we just added
	rewrite := map[uint8]uint8{
		jumpNextChannelPlaceholder: uint8(len(program)) - 2,
		jumpRejectPlaceholder:      uint8(len(program)) - 2,
		jumpAcceptPlaceholder:      uint8(len(program)) - 1,
	}
	// Now we back-track through the program to find the placeholders and
	// rewrite those to a real jump to the next channel (or the reject instruction).
	for i := uint8(len(program)) - 1; i > 0; i-- {
		switch instr := program[i].(type) {
		case bpf.JumpIf:
			if offset, ok := rewrite[instr.SkipTrue]; ok {
				// the skip values are off by one from an index perspective
				instr.SkipTrue = offset - i - 1
				program[i] = instr
			}
		case bpf.LoadAbsolute:
			// Each channel matching starts with a load of the first byte, uint16 or uint32 in channel.
			// If it's not, it's not the start of a the channel name
			if instr.Off == offsetChannel {
				rewrite[jumpNextChannelPlaceholder] = i
			}
		}
	}
	return program
}

// shortProtoMessageFilter accepts LCM short messages where the channel equals any of the proto message names.
func shortProtoMessageFilter(msgs ...proto.Message) []bpf.Instruction {
	channels := make([]string, len(msgs))
	for i, msg := range msgs {
		channels[i] = proto.MessageName(msg)
	}
	return shortMessageChannelFilter(channels...)
}
