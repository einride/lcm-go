package lcm

import "golang.org/x/net/bpf"

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
		for i := 0; i < len(channel)+1; /* null byte */ i++ {
			// check if the i:th byte matches, skip to next channel if not
			currByteIndex := indexOfUDPPayload + indexOfChannel + uint32(i)
			// 2 remaining instructions per byte plus the return for accepting
			remainingInstructions := (uint8(len(channel))-uint8(i))*2 + 1
			var currByte uint32
			if i < len(channel) {
				currByte = uint32(channel[i])
			}
			program = append(program,
				bpf.LoadAbsolute{Off: currByteIndex, Size: 1},
				bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: currByte, SkipTrue: remainingInstructions},
			)
		}
		// channel match, accept package
		program = append(program, bpf.RetConstant{Val: lengthOfLargestUDPMessage})
	}
	// no channel match, reject package
	program = append(program, bpf.RetConstant{Val: 0})
	return program
}
