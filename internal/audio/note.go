package audio

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/comprofix/gameskeleton/internal/config"
)

// Note is one step of a part: a pitch held for a time. Hz 0 is a rest.
type Note struct {
	Hz  float64
	Dur time.Duration
}

// Ticks is the length of n game ticks, so tables can be written in ticks.
func Ticks(n int) time.Duration {
	return time.Duration(n) * time.Second / config.TPS
}

// N is the named note (e.g. "C4", "F#5", "Bb3") held for the given ticks.
// It panics on a bad name, since note tables are fixed at build time.
func N(name string, ticks int) Note {
	hz, err := Pitch(name)
	if err != nil {
		panic(err)
	}
	return Note{Hz: hz, Dur: Ticks(ticks)}
}

// Rest is silence for the given ticks.
func Rest(ticks int) Note { return Note{Dur: Ticks(ticks)} }

// Notes reads notes written as "NAME:TICKS" separated by spaces, e.g.
// "C4:12 E4:12 -:24", where "-" is a rest. Several strings, such as one per
// bar, are joined. It panics on a mistake, since note tables are fixed at
// build time.
func Notes(bars ...string) []Note {
	var out []Note
	for _, bar := range bars {
		for _, f := range strings.Fields(bar) {
			name, ticks, ok := strings.Cut(f, ":")
			n, err := strconv.Atoi(ticks)
			if !ok || err != nil || n <= 0 {
				panic(fmt.Sprintf("audio: bad note %q", f))
			}
			if name == "-" {
				out = append(out, Rest(n))
			} else {
				out = append(out, N(name, n))
			}
		}
	}
	return out
}

// Pitch returns the frequency of a named note in equal temperament, with A4
// at 440 Hz: a letter A–G, an optional # or b, and an octave 0–9.
func Pitch(name string) (float64, error) {
	semis := map[byte]int{'C': -9, 'D': -7, 'E': -5, 'F': -4, 'G': -2, 'A': 0, 'B': 2}
	if len(name) < 2 {
		return 0, fmt.Errorf("audio: bad note %q", name)
	}
	s, ok := semis[name[0]]
	if !ok {
		return 0, fmt.Errorf("audio: bad note %q", name)
	}
	rest := name[1:]
	switch rest[0] {
	case '#':
		s, rest = s+1, rest[1:]
	case 'b':
		s, rest = s-1, rest[1:]
	}
	oct, err := strconv.Atoi(rest)
	if err != nil || oct < 0 || oct > 9 {
		return 0, fmt.Errorf("audio: bad note %q", name)
	}
	s += (oct - 4) * 12
	return 440 * math.Pow(2, float64(s)/12), nil
}

// Length is how long the notes take to play.
func Length(notes []Note) time.Duration {
	var d time.Duration
	for _, n := range notes {
		d += n.Dur
	}
	return d
}

// Wave is a channel's waveform.
type Wave int

const (
	Pulse50  Wave = iota // square wave
	Pulse25              // thinner, brighter
	Pulse12              // thin and nasal (12.5% duty)
	Triangle             // soft, for bass
	Noise                // for drums and hits; Hz sets how fast it changes
)

// Voice is how a part sounds: its wave and volume envelope. The level rises
// over Attack, falls over Decay to Sustain while the note holds, and falls
// to silence over Release once it ends. With no Decay the note holds at
// full level. A Volume of 0 means 1.
type Voice struct {
	Wave                   Wave
	Attack, Decay, Release time.Duration
	Sustain                float64 // 0–1
	Volume                 float64 // 0–1
}

// Part is one channel's line of a tune or effect.
type Part struct {
	Voice Voice
	Notes []Note
}

// Tune is music: part i plays on channel i. A looping tune's parts loop on
// their own, so keep them the same length. In the Speaker style only the
// first part plays.
type Tune struct {
	Name  string
	Parts []Part
	Loop  bool
}

// Effect is a sound effect: one part that takes over a channel while it
// plays. A new effect replaces one already on its channel only if its
// Priority is at least as high. In the Speaker style every effect uses the
// one channel.
type Effect struct {
	Name     string
	Channel  int
	Priority int
	Part     Part
}

// Bank lists the game's sounds, for the sound test.
type Bank struct {
	Tunes   []*Tune
	Effects []*Effect
}

// Join puts banks together.
func Join(banks ...Bank) Bank {
	var b Bank
	for _, x := range banks {
		b.Tunes = append(b.Tunes, x.Tunes...)
		b.Effects = append(b.Effects, x.Effects...)
	}
	return b
}

// Check reports the first mistake in the bank's tables: a missing or
// repeated name, an empty part, an effect on a channel Chip4 doesn't have,
// or a looping tune whose parts differ in length and so drift apart.
func (b Bank) Check() error {
	names := map[string]bool{}
	name := func(n string) error {
		if n == "" {
			return fmt.Errorf("audio: a sound has no name")
		}
		if names[n] {
			return fmt.Errorf("audio: two sounds are named %q", n)
		}
		names[n] = true
		return nil
	}
	for _, t := range b.Tunes {
		if err := name(t.Name); err != nil {
			return err
		}
		for i, p := range t.Parts {
			if Length(p.Notes) <= 0 {
				return fmt.Errorf("audio: tune %q part %d is empty", t.Name, i)
			}
			if t.Loop && partSamples(p) != partSamples(t.Parts[0]) {
				return fmt.Errorf("audio: looping tune %q part %d lasts %v, part 0 lasts %v",
					t.Name, i, Length(p.Notes), Length(t.Parts[0].Notes))
			}
		}
	}
	for _, e := range b.Effects {
		if err := name(e.Name); err != nil {
			return err
		}
		if Length(e.Part.Notes) <= 0 {
			return fmt.Errorf("audio: effect %q is empty", e.Name)
		}
		if e.Channel < 0 || e.Channel > 3 {
			return fmt.Errorf("audio: effect %q is on channel %d, want 0–3", e.Name, e.Channel)
		}
	}
	return nil
}

// partSamples is how many samples the part plays for, as the synth counts
// them: a tick is not a whole number of nanoseconds, so durations added up
// note by note can be a nanosecond out.
func partSamples(p Part) int {
	n := 0
	for _, note := range p.Notes {
		n += samples(note.Dur)
	}
	return n
}
