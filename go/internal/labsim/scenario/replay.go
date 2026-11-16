package scenario

import (
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
)

// replayRange checks a replay's fields, reads its file (relative to dir) and
// returns the lowest and highest value it plays.
func replayRange(b *Behaviour, dir, where string, out *faults) (low, high float64, ok bool) {
	before := len(out.list)
	b.series = nil
	if b.Speed != nil && *b.Speed <= 0 {
		out.add(where, msgReplaySpeed)
	}
	if b.End != "" && b.End != EndHoldLast && b.End != EndLoop {
		out.add(where, msgReplayEnd, b.End)
	}
	if b.StartOffsetMS < 0 {
		out.add(where, msgReplayOffset)
	}
	if b.File == "" {
		out.add(where, msgReplayFile)
		return 0, 0, false
	}
	s, err := readReplayFile(b.File, dir)
	if err != nil {
		out.add(where, msgReplayCannot, b.File, err)
		return 0, 0, false
	}
	if span := s.span(); time.Duration(b.StartOffsetMS)*time.Millisecond > span {
		out.add(where, msgReplayPast, b.StartOffsetMS, span.Milliseconds())
	}
	if len(out.list) != before {
		return 0, 0, false
	}
	b.series = s
	return slices.Min(s.Values), slices.Max(s.Values), true
}

func readReplayFile(name, dir string) (*Series, error) {
	path := name
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	return ReadSeries(fh)
}

// Series is the replay file Validate read; nil for another behaviour or a
// replay with a fault.
func (b *Behaviour) Series() *Series { return b.series }

// span is the time from the first record to the last.
func (s *Series) span() time.Duration {
	return s.Times[len(s.Times)-1].Sub(s.Times[0])
}

// replayGenerator plays the series Validate read.
func replayGenerator(b *Behaviour) model.Generator {
	s := b.series
	at := make([]time.Duration, len(s.Times))
	for i, t := range s.Times {
		at[i] = t.Sub(s.Times[0])
	}
	speed := 1.0
	if b.Speed != nil {
		speed = *b.Speed
	}
	return model.Replay(at, s.Values, speed, millis(b.StartOffsetMS), b.End == EndLoop)
}
