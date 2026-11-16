package model

import (
	"math"
	"math/rand/v2"
	"time"
)

// Generator gives a point's value at a time since the simulator started.
type Generator interface {
	At(elapsed time.Duration) float64
}

type constant float64

func (c constant) At(time.Duration) float64 { return float64(c) }

type ramp struct {
	min, max float64
	period   time.Duration
}

// Ramp climbs linearly from min to max over period, then starts again at min.
func Ramp(min, max float64, period time.Duration) Generator {
	return ramp{min: min, max: max, period: period}
}

func (r ramp) At(elapsed time.Duration) float64 {
	part := float64(elapsed%r.period) / float64(r.period)
	return r.min + (r.max-r.min)*part
}

type sine struct {
	center, amplitude float64
	period            time.Duration
}

// Sine swings amplitude above and below center once per period, starting at
// center on its way up.
func Sine(center, amplitude float64, period time.Duration) Generator {
	return sine{center: center, amplitude: amplitude, period: period}
}

func (s sine) At(elapsed time.Duration) float64 {
	part := float64(elapsed%s.period) / float64(s.period)
	return s.center + s.amplitude*math.Sin(2*math.Pi*part)
}

// walk is a random walk; it keeps the last step it computed, so it must be read
// under its device's lock.
type walk struct {
	rng      *rand.Rand
	value    float64
	step     float64
	interval time.Duration
	min, max float64
	done     int64 // steps already taken
}

// RandomWalk starts at start and, once per interval, moves by a uniform amount
// of at most step either way, held within min and max. The seed fixes the whole
// sequence: the value at a given time is the same in every run, however often
// and whenever it is read.
func RandomWalk(seed uint64, start, step float64, interval time.Duration, min, max float64) Generator {
	return &walk{rng: rand.New(rand.NewPCG(seed, 0)), value: start, step: step, interval: interval, min: min, max: max}
}

func (w *walk) At(elapsed time.Duration) float64 {
	for target := int64(elapsed / w.interval); w.done < target; w.done++ {
		next := w.value + (2*w.rng.Float64()-1)*w.step
		w.value = math.Min(w.max, math.Max(w.min, next))
	}
	return w.value
}
