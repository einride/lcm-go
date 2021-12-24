package lcmlogplayer

import (
	"context"
	"fmt"
	"io"
	"time"

	"go.einride.tech/lcm/lcmlog"
)

type Transmitter interface {
	Transmit(ctx context.Context, channel string, data []byte) error
}

type LogPlayer struct {
	f           io.ReadSeekCloser
	dur         time.Duration
	speedFactor float64
	transmitter Transmitter
}

func New(
	file io.ReadSeekCloser,
	dur time.Duration,
	speedFactor float64,
	transmitter Transmitter,
) *LogPlayer {
	return &LogPlayer{
		f:           file,
		dur:         dur,
		speedFactor: speedFactor,
		transmitter: transmitter,
	}
}

func (p *LogPlayer) Play(ctx context.Context, progressCallback func(messageNumber int)) (int, error) {
	scanner := lcmlog.NewScanner(p.f)
	var previousMsg lcmlog.Message
	firstMessage := true
	noMessages := 0
	skippedMessages := 0
	for scanner.Scan() {
		if firstMessage {
			previousMsg = *scanner.Message()
			firstMessage = false
			continue
		}
		newMessage := *scanner.Message()
		tDiff := newMessage.Timestamp.Sub(previousMsg.Timestamp)
		progressCallback(noMessages)
		noMessages++
		if tDiff > p.dur {
			skippedMessages++
			previousMsg = newMessage
			continue
		}
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return 0, fmt.Errorf("play: %w", err)
			}
			return 0, nil
		case <-time.After(tDiff / time.Duration(p.speedFactor)):
			err := p.transmitter.Transmit(ctx, previousMsg.Channel, previousMsg.Data)
			if err != nil {
				return 0, fmt.Errorf("play: %w", err)
			}
			previousMsg = newMessage
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("play: %w", err)
	}
	if err := p.resetFilePosition(); err != nil {
		return 0, fmt.Errorf("play: %w", err)
	}
	return skippedMessages, nil
}

func (p *LogPlayer) resetFilePosition() error {
	if _, err := p.f.Seek(0, 0); err != nil {
		return fmt.Errorf("reset file position: %w", err)
	}
	return nil
}

// GetLength returns the duration of the log and number of messages.
func (p *LogPlayer) GetLength() (time.Duration, int, error) {
	if err := p.resetFilePosition(); err != nil {
		return 0, 0, fmt.Errorf("get length: %w", err)
	}
	lengthScanner := lcmlog.NewScanner(p.f)
	if !lengthScanner.Scan() {
		return 0, 0, fmt.Errorf("get length: %w", lengthScanner.Err())
	}
	firstTimestamp := lengthScanner.Message().Timestamp
	noMessages := 1
	for lengthScanner.Scan() {
		noMessages++
	}
	timeLength := lengthScanner.Message().Timestamp.Sub(firstTimestamp)
	if err := p.resetFilePosition(); err != nil {
		return 0, 0, fmt.Errorf("get length: %w", err)
	}
	return timeLength, noMessages, nil
}
