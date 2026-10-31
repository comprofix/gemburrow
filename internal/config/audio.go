package config

// AudioStyle is how the synth sounds.
type AudioStyle string

const (
	// Chip4 is four channels (2 pulse, triangle, noise); music keeps time
	// under a sound effect.
	Chip4 AudioStyle = "chip4"
	// Speaker is one square-wave channel, like an old PC speaker; a sound
	// effect interrupts the music, which then carries on where it stopped.
	Speaker AudioStyle = "speaker"
)

// Audio output.
const (
	AudioSampleRate = 44100
	AudioBufferMs   = 50 // output buffer; keeps effects close to their events

	// AudioScheduleMs is the fixed delay between a sound's game tick and
	// when it plays, so sounds land exactly on their tick despite the
	// device reading audio ahead in chunks. It widens up to
	// AudioScheduleMaxMs if the device reads further ahead.
	AudioScheduleMs    = 60
	AudioScheduleMaxMs = 250

	// ChannelLevel is one Chip4 channel's peak level at full volume, out
	// of 1, leaving headroom for four channels together. SpeakerLevel is
	// the Speaker style's one channel, which needs no headroom.
	ChannelLevel = 0.22
	SpeakerLevel = 0.5
)

// Synth tuning.
const (
	// AudioRampMs is the shortest attack or release, so notes never start
	// or stop with a click.
	AudioRampMs = 1.0
	// AudioDCBlockHz is the cutoff of the filter that keeps the mix centred
	// on zero (thin pulse waves are lopsided).
	AudioDCBlockHz = 10.0
)
