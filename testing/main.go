package main

import (
	"log"
	"os"
	"time"

	"github.com/1lann/fxxk"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

func main() {
	f, err := os.Open("sengoku.mp3")
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	c := make(chan struct{})
	bpp := beep.StreamerFunc(func(samples [][2]float64) (int, bool) {
		c <- struct{}{}
		return streamer.Stream(samples)
	})

	st := fxxk.NewFullStreamer(fxxk.NewRealtimeStream(bpp, format.SampleRate, fxxk.BufferConfig{
		Target:   time.Millisecond * 20,
		Min:      time.Millisecond * 10,
		Max:      time.Millisecond * 100,
		Overruns: false,
	}))

	log.Println(speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)))
	speaker.Play(st)

	aft := time.After(time.Second * 5)

big:
	for {
		select {
		case <-aft:
			break big
		case <-c:
			continue
		}
	}

	time.Sleep(time.Second * 3)

	for {
		select {
		case <-c:
			continue
		}
	}

	select {}
}
