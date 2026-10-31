package audio

import (
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/comprofix/gameskeleton/internal/config"
)

// Synth plays tunes and effects on a set of channels in one of the styles:
//   - Chip4: four channels. Tune part i plays on channel i. An effect takes
//     over its channel while it plays; the music underneath keeps time and
//     comes back in step.
//   - Speaker: one channel, square waves only (noise still works, as a
//     speaker can click randomly). An effect interrupts the music, which
//     then carries on from where it stopped.
//
// The game thread sends commands; the audio thread pulls samples through
// Read. All methods are safe for concurrent use, and a nil *Synth ignores
// every call, so tests can run scenes without sound.
type Synth struct {
	mu    sync.Mutex
	style config.AudioStyle
	ch    []channel
	tune  *Tune // the tune last started, as the game thread sees it

	// Scheduling (see Now): commands wait in queue until the sample count
	// pos reaches their time. offset maps game time onto samples.
	queue   []command
	pos     int64
	timed   bool
	latency int64 // the current delay, in samples; it widens to cover the device's read-ahead
	chunk   int64 // the size of the device's last read, in samples
	stamp   int64
	offset  int64

	level                  float64 // each channel's peak level, by style
	enabled                bool
	master, musicVol, sfxV float64
	dcIn, dcOut            float64 // DC-blocking filter state
}

// NewSynth returns an enabled synth in the given style at full volume.
func NewSynth(style config.AudioStyle) *Synth {
	s := &Synth{style: style, ch: make([]channel, 4), level: config.ChannelLevel, enabled: true, master: 1, musicVol: 1, sfxV: 1}
	if style == config.Speaker {
		s.ch, s.level = s.ch[:1], config.SpeakerLevel
	}
	for i := range s.ch {
		s.ch[i].lfsr = 1
	}
	return s
}

// Channels is how many channels the synth has.
func (s *Synth) Channels() int {
	if s == nil {
		return 0
	}
	return len(s.ch)
}

// PlayTune starts a tune from its beginning, replacing the one playing. The
// tune must not be changed while it plays.
func (s *Synth) PlayTune(t *Tune) {
	if s == nil || t == nil {
		return
	}
	s.mu.Lock()
	s.tune = t
	s.mu.Unlock()
	s.send(command{kind: playTune, tune: t})
}

// StopTune stops the music.
func (s *Synth) StopTune() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.tune = nil
	s.mu.Unlock()
	s.send(command{kind: stopTune})
}

// Tune is the tune last started, or nil after StopTune or Stop. Scenes use
// it to leave a tune playing rather than restart it.
func (s *Synth) Tune() *Tune {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tune
}

// PlayEffect starts an effect on its channel, unless an effect of higher
// priority is still playing there.
func (s *Synth) PlayEffect(e *Effect) {
	if s == nil || e == nil {
		return
	}
	s.send(command{kind: playEffect, effect: e})
}

// Stop ends the music and every effect.
func (s *Synth) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.tune = nil
	s.mu.Unlock()
	s.send(command{kind: stopAll})
}

// SetEnabled turns all sound on or off. The music keeps time while sound
// is off.
func (s *Synth) SetEnabled(on bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = on
}

// SetVolumes sets the master, music and effects volumes, each 0–1.
func (s *Synth) SetVolumes(master, music, sfx float64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.master, s.musicVol, s.sfxV = unit(master), unit(music), unit(sfx)
}

func unit(v float64) float64 {
	if !(v >= 0) {
		return 0
	}
	return min(v, 1)
}

// Now sets the game tick that the commands sent after it belong to. Once it
// has been called, each command plays at its own tick plus a fixed delay,
// exact to the sample, rather than whenever the device next reads audio.
// Without it, as in tests, commands apply at once.
func (s *Synth) Now(tick int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	const (
		perTick = config.AudioSampleRate / config.TPS
		base    = config.AudioScheduleMs * config.AudioSampleRate / 1000
		most    = config.AudioScheduleMaxMs * config.AudioSampleRate / 1000
	)
	at := tick * perTick
	if !s.timed {
		s.latency = base
	}
	// Anchor the game clock to the audio clock. The device reads audio
	// ahead in chunks by an amount we cannot know, so if a command would
	// already be late, widen the delay to cover it and re-anchor; this
	// settles after a moment. If commands would wait longer than the delay
	// plus one read, re-anchor too: the device started after the game, or
	// the clocks drifted, and every sound would play that much late.
	ahead := at + s.offset - s.pos
	if !s.timed || ahead < 0 || ahead > s.latency+max(s.chunk, perTick)+perTick {
		// A little late: the device reads further ahead than allowed for.
		// Very late means the game stalled, which needs no wider delay.
		if s.timed && ahead < 0 && -ahead <= most {
			s.latency = min(s.latency-ahead+base/2, most)
		}
		s.offset = s.pos + s.latency - at
		s.timed = true
	}
	s.stamp = at + s.offset
}

// send queues a command at the current game time, or applies it at once if
// the game has not set a time.
func (s *Synth) send(c command) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.timed {
		s.apply(c)
		return
	}
	c.at = s.stamp
	if n := len(s.queue); n > 0 {
		c.at = max(c.at, s.queue[n-1].at) // keep the order they were sent in
	}
	s.queue = append(s.queue, c)
}

// command is a queued change to the channels.
type command struct {
	at     int64 // the sample it applies at
	kind   commandKind
	tune   *Tune
	effect *Effect
}

type commandKind int

const (
	playTune commandKind = iota
	stopTune
	playEffect
	stopAll
)

// apply carries out a command.
func (s *Synth) apply(c command) {
	switch c.kind {
	case playTune:
		for i := range s.ch {
			if i < len(c.tune.Parts) {
				s.ch[i].music.start(&c.tune.Parts[i], c.tune.Loop)
			} else {
				s.ch[i].music.stop()
			}
		}
	case stopTune:
		for i := range s.ch {
			s.ch[i].music.stop()
		}
	case playEffect:
		e := c.effect
		ch := &s.ch[min(max(e.Channel, 0), len(s.ch)-1)]
		if ch.sfx.active() && ch.prio > e.Priority {
			return
		}
		ch.sfx.start(&e.Part, false)
		ch.prio = e.Priority
	case stopAll:
		for i := range s.ch {
			s.ch[i].music.stop()
			s.ch[i].sfx.stop()
		}
	}
}

// Read fills p with 16-bit little-endian stereo samples at
// config.AudioSampleRate. The stream never ends.
func (s *Synth) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dc := math.Exp(-2 * math.Pi * config.AudioDCBlockHz / config.AudioSampleRate)
	n := len(p) / 4
	s.chunk = int64(n)
	for i := range n {
		for len(s.queue) > 0 && s.queue[0].at <= s.pos {
			s.apply(s.queue[0])
			s.queue = s.queue[1:]
		}
		s.pos++

		mix := 0.0
		for c := range s.ch {
			mix += s.ch[c].sample(s)
		}
		if !s.enabled {
			mix = 0
		}
		s.dcOut = mix - s.dcIn + dc*s.dcOut
		s.dcIn = mix
		x := min(max(s.dcOut*s.master, -1), 1)

		v := uint16(int16(math.Round(x * math.MaxInt16)))
		binary.LittleEndian.PutUint16(p[4*i:], v)
		binary.LittleEndian.PutUint16(p[4*i+2:], v)
	}
	return 4 * n, nil
}

// channel is one voice with a music track and an effect track. The effect
// sounds whenever it is playing.
type channel struct {
	music, sfx track
	prio       int    // the playing effect's priority
	last       *track // the track that sounded last sample
	fromSFX    bool   // the sounding note is the effect's

	voice Voice
	hz    float64
	phase float64 // 0–1
	lfsr  uint16  // noise generator
	stage stage
	level float64 // envelope, 0–1
	fall  float64 // release step per sample
}

// stage is where a channel's envelope is.
type stage int

const (
	off stage = iota
	attack
	decay
	hold
	release
)

// sample advances the channel one sample and returns its output.
func (c *channel) sample(s *Synth) float64 {
	var t *track
	switch {
	case c.sfx.active():
		t = &c.sfx
	case c.music.active():
		t = &c.music
	}
	if t != nil {
		fresh := t.take() || t != c.last
		c.last, c.fromSFX = t, t == &c.sfx
		if fresh {
			if hz := t.hz(); hz > 0 {
				c.noteOn(t.part.Voice, hz, s.style)
			} else {
				c.noteOff()
			}
		}
		t.step()
		// In Chip4 the music keeps time under an effect.
		if t == &c.sfx && s.style == config.Chip4 && c.music.active() {
			c.music.step()
		}
	} else if c.last != nil {
		c.noteOff()
		c.last = nil
	}
	// A note fading out after its track ended keeps its track's volume.
	vol := s.musicVol
	if c.fromSFX {
		vol = s.sfxV
	}

	c.envelope()
	if c.level == 0 {
		return 0
	}
	return c.wave() * c.level * c.voice.volume() * s.level * vol
}

// noteOn starts a note, from the envelope's current level so a retriggered
// note doesn't click.
func (c *channel) noteOn(v Voice, hz float64, style config.AudioStyle) {
	if style == config.Speaker && v.Wave != Noise {
		v.Wave = Pulse50
	}
	if c.level == 0 {
		c.phase = 0
	}
	c.voice, c.hz, c.stage = v, min(hz, config.AudioSampleRate*0.45), attack
}

// noteOff lets the note fade over its release.
func (c *channel) noteOff() {
	if c.stage == off || c.stage == release {
		return
	}
	c.stage = release
	c.fall = c.level / ramp(c.voice.Release)
}

// envelope advances the volume envelope one sample.
func (c *channel) envelope() {
	switch c.stage {
	case attack:
		c.level += 1 / ramp(c.voice.Attack)
		if c.level >= 1 {
			c.level, c.stage = 1, hold
			if c.voice.Decay > 0 {
				c.stage = decay
			}
		}
	case decay:
		sus := unit(c.voice.Sustain)
		c.level -= (1 - sus) / float64(max(samples(c.voice.Decay), 1))
		if c.level <= sus {
			c.level, c.stage = sus, hold
		}
	case release:
		c.level -= c.fall
		if c.level <= 0 {
			c.level, c.stage = 0, off
		}
	}
}

// ramp is a length in samples, at least config.AudioRampMs.
func ramp(d time.Duration) float64 {
	return max(float64(samples(d)), config.AudioRampMs*config.AudioSampleRate/1000)
}

// wave returns the waveform's value, -1 to 1, and advances its phase.
func (c *channel) wave() float64 {
	dt := c.hz / config.AudioSampleRate
	var v float64
	switch c.voice.Wave {
	case Pulse50:
		v = pulse(c.phase, dt, 0.5)
	case Pulse25:
		v = pulse(c.phase, dt, 0.25)
	case Pulse12:
		v = pulse(c.phase, dt, 0.125)
	case Triangle:
		v = 4*math.Abs(c.phase-0.5) - 1
	case Noise:
		v = -1
		if c.lfsr&1 != 0 {
			v = 1
		}
	}
	c.phase += dt
	if c.phase >= 1 {
		c.phase -= 1
		if c.voice.Wave == Noise {
			// A 15-bit shift register: noise that repeats only every
			// 32767 steps.
			bit := (c.lfsr ^ c.lfsr>>1) & 1
			c.lfsr = c.lfsr>>1 | bit<<14
		}
	}
	return v
}

// pulse is a pulse wave, high for the first duty of each cycle,
// band-limited with polyBLEP so high notes don't alias into harshness. dt
// is the phase step per sample.
func pulse(phase, dt, duty float64) float64 {
	v := -1.0
	if phase < duty {
		v = 1
	}
	fall := phase - duty
	if fall < 0 {
		fall += 1
	}
	return v + polyBLEP(phase, dt) - polyBLEP(fall, dt)
}

// polyBLEP is the correction for a unit step at phase 0.
func polyBLEP(t, dt float64) float64 {
	switch {
	case t < dt:
		t /= dt
		return t + t - t*t - 1
	case t > 1-dt:
		t = (t - 1) / dt
		return t*t + t + t + 1
	}
	return 0
}

func (v Voice) volume() float64 {
	if v.Volume == 0 {
		return 1
	}
	return unit(v.Volume)
}

// track plays a part, one sample at a time.
type track struct {
	part  *Part
	loop  bool
	i     int  // current note
	left  int  // samples left in it
	fresh bool // the current note has not sounded yet
}

func (t *track) active() bool { return t.part != nil }

func (t *track) hz() float64 { return t.part.Notes[t.i].Hz }

// take reports whether the current note has just started, once.
func (t *track) take() bool {
	f := t.fresh
	t.fresh = false
	return f
}

func (t *track) start(p *Part, loop bool) {
	t.part, t.loop, t.i = p, loop, 0
	if len(p.Notes) == 0 {
		t.stop()
		return
	}
	t.left = samples(p.Notes[0].Dur)
	t.fresh = true
	t.skipEmpty()
}

func (t *track) stop() { t.part = nil }

// step advances one sample.
func (t *track) step() {
	t.left--
	t.skipEmpty()
}

// skipEmpty moves past finished notes, looping or ending the track. A
// looping part with no length at all is stopped rather than spun on.
func (t *track) skipEmpty() {
	for wrapped := false; t.left <= 0; {
		t.i++
		if t.i == len(t.part.Notes) {
			if !t.loop || wrapped {
				t.stop()
				return
			}
			t.i, wrapped = 0, true
		}
		t.left = samples(t.part.Notes[t.i].Dur)
		t.fresh = true
	}
}

func samples(d time.Duration) int {
	return int(math.Round(d.Seconds() * config.AudioSampleRate))
}
