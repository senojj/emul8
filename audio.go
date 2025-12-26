package emul8

import (
	"context"
	"sync/atomic"

	"github.com/go-audio/audio"
	"github.com/go-audio/generator"
	"github.com/gordonklaus/portaudio"
	"golang.org/x/sync/errgroup"
)

const (
	bufferSize int     = 512
	note       float64 = 440.0
)

var (
	format = audio.FormatMono44100
)

type Beep struct {
	g       errgroup.Group
	beeping atomic.Bool
}

func (b *Beep) Start(ctx context.Context) error {
	if b.beeping.Load() {
		return nil
	}
	b.beeping.Store(true)

	err := portaudio.Initialize()
	if err != nil {
		return err
	}

	buffer := &audio.FloatBuffer{
		Data:   make([]float64, bufferSize),
		Format: format,
	}

	osc := generator.NewOsc(generator.WaveSine, note, buffer.Format.SampleRate)
	osc.Amplitude = 1

	b.g.Go(func() error {
		defer func() {
			_ = portaudio.Terminate()
		}()

		out := make([]float32, bufferSize)

		stream, err := portaudio.OpenDefaultStream(0, 1, 44100, len(out), &out)
		if err != nil {
			return err
		}
		defer func() {
			_ = stream.Close()
		}()

		if err := stream.Start(); err != nil {
			return err
		}
		defer func() {
			_ = stream.Stop()
		}()

		for b.beeping.Load() && ctx.Err() == nil {
			if err := osc.Fill(buffer); err != nil {
				return err
			}

			f64Tof32(out, buffer.Data)

			if err := stream.Write(); err != nil {
				return err
			}
		}

		return nil
	})

	return nil
}

func (b *Beep) Stop() error {
	if !b.beeping.Load() {
		return nil
	}
	b.beeping.Store(false)
	return b.g.Wait()
}

func f64Tof32(dst []float32, src []float64) {
	for i := range src {
		dst[i] = float32(src[i])
	}
}
