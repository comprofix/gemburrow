package play

import "github.com/comprofix/gemburrow/internal/audio"

// Placeholder sounds, replaced with the game's own later. Channels follow the
// shared layout: 0 lead, 1 harmony, 2 bass, 3 noise.
var (
	// Dig is a cell of earth dug.
	Dig = &audio.Effect{Name: "dig", Channel: 3, Priority: 1, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(5), Volume: 0.4},
		Notes: []audio.Note{{Hz: 1800, Dur: audio.Ticks(5)}},
	}}
	// Jewel is a jewel collected.
	Jewel = &audio.Effect{Name: "jewel", Channel: 1, Priority: 2, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(8), Sustain: 0.4, Release: audio.Ticks(3), Volume: 0.8},
		Notes: audio.Notes("E6:3 B6:8"),
	}}
	// LevelClear is the last jewel of a level collected.
	LevelClear = &audio.Effect{Name: "level clear", Channel: 0, Priority: 5, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse50, Decay: audio.Ticks(16), Sustain: 0.5, Release: audio.Ticks(8), Volume: 0.8},
		Notes: audio.Notes("D5:6 A5:6 D6:24"),
	}}
)

// Sounds are the game's sounds, for the sound test.
var Sounds = audio.Bank{Effects: []*audio.Effect{Dig, Jewel, LevelClear}}
