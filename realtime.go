// Utilities that beep should have came with.
package fxxk

import (
	"sync"
	"time"

	"github.com/faiface/beep"
)

type RealtimeStream struct {
	streamer   beep.Streamer
	sampleRate beep.SampleRate

	lastRead        time.Time
	zeroSamplesSent int
	overruns        bool
	isUnderrun      bool

	target int
	min    int
	max    int

	err error

	buffer [][2]float64
	mutex  *sync.Mutex
}

func (r *RealtimeStream) pump() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for {
		bufSize := r.max - len(r.buffer)
		overrun := false
		if len(r.buffer) < r.min {
			bufSize = r.target - len(r.buffer)
		}

		if bufSize <= 0 {
			bufSize = r.min
			overrun = true
		}

		if overrun && !r.overruns {
			r.mutex.Unlock()
			time.Sleep(10 * time.Millisecond)
			r.mutex.Lock()
			continue
		}

		buf := make([][2]float64, bufSize)

		r.mutex.Unlock()
		n, err := readFull(r.streamer, buf)
		r.mutex.Lock()

		r.isUnderrun = false
		r.buffer = append(r.buffer, buf[:n]...)
		if err != nil {
			r.err = err
			return
		}

		// overrun and overruns are enabled
		if overrun {
			r.buffer = r.buffer[bufSize:]
		}
	}
}

type BufferConfig struct {
	Target   time.Duration
	Min      time.Duration
	Max      time.Duration
	Overruns bool
}

func NewRealtimeStream(streamer beep.Streamer, sampleRate beep.SampleRate, bufferConfig BufferConfig) *RealtimeStream {
	r := &RealtimeStream{
		streamer:   streamer,
		sampleRate: sampleRate,
		overruns:   bufferConfig.Overruns,
		mutex:      new(sync.Mutex),
		target:     sampleRate.N(bufferConfig.Target),
		min:        sampleRate.N(bufferConfig.Min),
		max:        sampleRate.N(bufferConfig.Max),
	}
	go r.pump()
	return r
}

func (r *RealtimeStream) Stream(samples [][2]float64) (n int, ok bool) {
	r.mutex.Lock()
	if len(r.buffer) == 0 && r.err != nil {
		r.mutex.Unlock()
		return 0, false
	}

	// got plenty to stream, stream it!
	if len(r.buffer) > r.min {
		r.zeroSamplesSent = 0
		d := len(r.buffer) - r.min
		if len(samples) < d {
			d = len(samples)
		}
		copy(samples, r.buffer[:d])
		r.buffer = r.buffer[d:]
		r.lastRead = time.Now().Add(r.sampleRate.D(d))

		r.mutex.Unlock()
		return d, true
	}

	// nothing to stream, see if there is catch up
	if !r.isUnderrun {
		// this is the first time, we can add tolerance, wait until target buffer length.
		waitTime := r.sampleRate.D(r.target-len(r.buffer)) - time.Since(r.lastRead)
		ds := r.sampleRate.D(len(samples))
		if ds < waitTime {
			waitTime = ds
		}

		r.mutex.Unlock()
		time.Sleep(waitTime)
		r.mutex.Lock()

		if len(r.buffer) <= r.min {
			// this is bad!! we've ran out of data
			r.isUnderrun = true
		}

		r.mutex.Unlock()
		return r.Stream(samples)
	} else {
		// this is not the first time :(
		// is there any buffer left? if so use that
		n := len(r.buffer)
		if n > 0 {
			if len(samples) < n {
				n = len(samples)
			}

			copy(samples, r.buffer[:n])
			r.buffer = r.buffer[n:]

			r.mutex.Unlock()
			return n, true
		}

		// calculate how many zero samples we need to prepare.
		samplesToSend := int(r.sampleRate.N(time.Since(r.lastRead)) - r.zeroSamplesSent)
		if samplesToSend <= int(r.sampleRate/500) {
			// oh we're doing this super quickly! too quick 4 me

			r.mutex.Unlock()
			time.Sleep(2 * time.Millisecond)
			return r.Stream(samples)
		}

		if len(samples) < samplesToSend {
			samplesToSend = len(samples)
		}

		r.zeroSamplesSent += samplesToSend
		for i := 0; i < samplesToSend; i++ {
			samples[i][0], samples[i][1] = 0, 0
		}

		r.mutex.Unlock()
		return samplesToSend, true
	}
}

func (r *RealtimeStream) Err() error {
	return r.err
}
