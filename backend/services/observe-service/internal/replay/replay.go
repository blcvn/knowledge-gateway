package replay

import (
	"context"
	"time"
)

type PlaybackConfig struct {
	Speed      float64 // 1.0 = realtime, 2.0 = 2x speed
	StartIndex int
}

type PlaybackEvent struct {
	Entry     TimelineEntry
	Progress  float64 // 0.0 - 1.0
	Remaining int     // events remaining
}

// Playback streams timeline events with time delays
// Returns a channel of PlaybackEvents; caller manages context cancellation
func Playback(ctx context.Context, timeline Timeline, cfg PlaybackConfig) <-chan PlaybackEvent {
	ch := make(chan PlaybackEvent, 10)

	go func() {
		defer close(ch)
		entries := timeline.Entries
		if cfg.StartIndex > 0 {
			entries = entries[cfg.StartIndex:]
		}
		speed := cfg.Speed
		if speed <= 0 {
			speed = 1.0
		}

		for i, entry := range entries {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Simulate time delay between events
			if i > 0 && entry.Duration > 0 {
				delay := time.Duration(float64(entry.Duration) / speed)
				if delay > 30*time.Second {
					delay = 30 * time.Second
				} // cap delay
				timer := time.NewTimer(delay)
				select {
				case <-timer.C:
				case <-ctx.Done():
					timer.Stop()
					return
				}
			}

			ch <- PlaybackEvent{
				Entry:     entry,
				Progress:  float64(i+1) / float64(len(entries)),
				Remaining: len(entries) - i - 1,
			}
		}
	}()

	return ch
}
