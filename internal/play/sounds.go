package play

import "github.com/comprofix/gemburrow/internal/audio"

// The game's sound effects. Channels follow the shared layout: 0 lead,
// 1 harmony, 2 bass, 3 noise.
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
	// JewelRun is a run of jewels completed.
	JewelRun = &audio.Effect{Name: "jewel run", Channel: 1, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(8), Sustain: 0.4, Release: audio.Ticks(3), Volume: 0.8},
		Notes: audio.Notes("E6:3 G6:3 B6:3 E7:10"),
	}}
	// LevelClear is the last jewel of a level collected.
	LevelClear = &audio.Effect{Name: "level clear", Channel: 0, Priority: 5, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse50, Decay: audio.Ticks(16), Sustain: 0.5, Release: audio.Ticks(8), Volume: 0.8},
		Notes: audio.Notes("D5:6 A5:6 D6:24"),
	}}
	// Fired is the dwarf firing.
	Fired = &audio.Effect{Name: "shot", Channel: 1, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(10), Release: audio.Ticks(2), Volume: 0.7},
		Notes: audio.Notes("A6:3 E6:3 A5:5"),
	}}
	// GoblinDown is a goblin destroyed.
	GoblinDown = &audio.Effect{Name: "goblin down", Channel: 3, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(16), Release: audio.Ticks(2), Volume: 0.8},
		Notes: []audio.Note{{Hz: 1200, Dur: audio.Ticks(16)}},
	}}
	// Caught is the blow as the dwarf dies. It is on the noise channel, to leave
	// the funeral march's channels alone.
	Caught = &audio.Effect{Name: "caught", Channel: 3, Priority: 5, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(14), Release: audio.Ticks(2), Volume: 0.8},
		Notes: []audio.Note{{Hz: 500, Dur: audio.Ticks(14)}},
	}}
	// LifeGained is an extra life.
	LifeGained = &audio.Effect{Name: "extra life", Channel: 0, Priority: 5, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(12), Sustain: 0.5, Release: audio.Ticks(6), Volume: 0.8},
		Notes: audio.Notes("G5:5 C6:5 E6:5 G6:16"),
	}}
	// SackShake is a sack starting to wobble, the warning before it falls
	// (DESIGN §8).
	SackShake = &audio.Effect{Name: "sack wobble", Channel: 2, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(10), Sustain: 0.6, Release: audio.Ticks(4), Volume: 0.9},
		Notes: audio.Notes("C3:5 D3:5 C3:5 D3:5 C3:5 D3:5"),
	}}
	// SackThud is a sack landing whole.
	SackThud = &audio.Effect{Name: "sack thud", Channel: 3, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(10), Release: audio.Ticks(2), Volume: 0.8},
		Notes: []audio.Note{{Hz: 300, Dur: audio.Ticks(10)}},
	}}
	// SackBurst is a sack bursting into gold.
	SackBurst = &audio.Effect{Name: "sack burst", Channel: 1, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(10), Sustain: 0.3, Release: audio.Ticks(4), Volume: 0.8},
		Notes: audio.Notes("C6:3 E6:3 G6:3 C7:8"),
	}}
	// MushroomUp is the mushroom appearing.
	MushroomUp = &audio.Effect{Name: "mushroom up", Channel: 1, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(10), Sustain: 0.4, Release: audio.Ticks(4), Volume: 0.8},
		Notes: audio.Notes("C5:4 G5:4 C6:10"),
	}}
	// MushroomTaken is the mushroom collected, starting the chase.
	MushroomTaken = &audio.Effect{Name: "mushroom taken", Channel: 0, Priority: 5, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(10), Sustain: 0.5, Release: audio.Ticks(4), Volume: 0.8},
		Notes: audio.Notes("A5:3 C6:3 E6:3 A6:12"),
	}}
	// GoblinEaten is a goblin eaten in the chase.
	GoblinEaten = &audio.Effect{Name: "goblin eaten", Channel: 1, Priority: 4, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse50, Decay: audio.Ticks(8), Release: audio.Ticks(3), Volume: 0.8},
		Notes: audio.Notes("E5:3 G5:3 E6:8"),
	}}
	// GoldTaken is gold collected.
	GoldTaken = &audio.Effect{Name: "gold", Channel: 1, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse50, Decay: audio.Ticks(10), Sustain: 0.4, Release: audio.Ticks(4), Volume: 0.8},
		Notes: audio.Notes("G6:4 B6:4 D7:10"),
	}}
)

// Sounds are the game's sounds, for the sound test.
var Sounds = audio.Bank{Tunes: []*audio.Tune{BurrowTune, ChaseTune, FuneralTune, ScoresTune}, Effects: []*audio.Effect{
	Dig, Jewel, JewelRun, LevelClear, Fired, GoblinDown, Caught, LifeGained, SackShake, SackThud, SackBurst, GoldTaken,
	MushroomUp, MushroomTaken, GoblinEaten,
}}
