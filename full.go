package fxxk

import (
	"io"

	"github.com/faiface/beep"
)

type readFullStreamer struct {
	streamer beep.Streamer
	err      error
}

// NewFullStreamer wraps a streamer to always read full.
func NewFullStreamer(streamer beep.Streamer) beep.Streamer {
	return &readFullStreamer{
		streamer: streamer,
	}
}

func (s *readFullStreamer) Stream(samples [][2]float64) (int, bool) {
	if s.err != nil {
		return 0, false
	}

	n, err := readFull(s.streamer, samples)
	if n == 0 {
		return 0, false
	}

	s.err = err

	return n, true
}

func (s *readFullStreamer) Err() error {
	return s.err
}

func readFull(streamer beep.Streamer, samples [][2]float64) (int, error) {
	var total int
	for total < len(samples) {
		n, ok := streamer.Stream(samples[total:])
		total += n
		if !ok {
			err := streamer.Err()
			if err != nil {
				return total, err
			}

			return total, io.EOF
		}
	}

	return total, nil
}
