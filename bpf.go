package lcm

import (
	"encoding/binary"

	"golang.org/x/net/bpf"
	"google.golang.org/protobuf/proto"
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
		remaining := []byte(channel)
		var i int
		for ; len(remaining) >= 4; i += 4 {
			program = append(program,
				bpf.LoadAbsolute{Off: offsetChannel + uint32(i), Size: 4},
				bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: binary.BigEndian.Uint32(remaining), SkipTrue: jumpNextChannelPlaceholder},
			)
			remaining = remaining[4:]
		}
		var val uint32
		var size int
		switch len(remaining) {
		case 0:
			size = 1
		case 1:
			val, size = uint32(remaining[0])<<8, 2
		case 2:
			val, size = uint32(remaining[1])<<8|uint32(remaining[0])<<16, 4
		case 3:
			val, size = uint32(remaining[2])<<8|uint32(remaining[1])<<16|uint32(remaining[0])<<24, 4
		}
		program = append(program, bpf.LoadAbsolute{Off: offsetChannel + uint32(i), Size: size})
		if len(remaining) == 2 {
			// When this happens we actually read 1 byte into the payload. So a packet with no payload
			// will be rejected too. But why would you do that?
			program = append(program, bpf.ALUOpConstant{Op: bpf.ALUOpShiftRight, Val: 0x8})
		}
		program = append(program,
			// Channel match, accept package.
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: val, SkipTrue: jumpAcceptPlaceholder},
			// Or if there is a query parameter accept the message as is.
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: val | '?', SkipTrue: jumpAcceptPlaceholder},
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
		channels[i] = string(msg.ProtoReflect().Descriptor().FullName())
	}
	return shortMessageChannelFilter(channels...)
}
