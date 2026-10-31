package demo

import "github.com/comprofix/gameskeleton/internal/audio"

// Dot Dash's sounds. Channels follow the shared tune layout (see
// scene.TitleTune): 0 lead, 1 harmony, 2 bass, 3 noise.
var (
	// Step is the player moving one cell.
	Step = &audio.Effect{Name: "step", Channel: 3, Priority: 1, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(3), Volume: 0.5},
		Notes: []audio.Note{{Hz: 6000, Dur: audio.Ticks(3)}},
	}}
	// Pickup is a dot collected.
	Pickup = &audio.Effect{Name: "pickup", Channel: 1, Priority: 2, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(8), Sustain: 0.4, Release: audio.Ticks(3), Volume: 0.8},
		Notes: audio.Notes("G6:3 C7:8"),
	}}
	// Bump is the player hitting a hazard.
	Bump = &audio.Effect{Name: "bump", Channel: 3, Priority: 3, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(14), Release: audio.Ticks(2)},
		Notes: []audio.Note{{Hz: 900, Dur: audio.Ticks(14)}},
	}}
	// GameOver is the end of a game.
	GameOver = &audio.Effect{Name: "game over", Channel: 0, Priority: 5, Part: audio.Part{
		Voice: audio.Voice{Wave: audio.Pulse50, Decay: audio.Ticks(20), Sustain: 0.5, Release: audio.Ticks(12), Volume: 0.8},
		Notes: audio.Notes("G5:10 E5:10 C5:10 -:4 G4:36"),
	}}
)

// DashTune loops during play: four bars of Am, F, C, G.
var DashTune = &audio.Tune{Name: "dot dash", Loop: true, Parts: []audio.Part{
	{Voice: lead, Notes: audio.Notes(
		"A4:12 C5:12 E5:12 A5:12 G5:12 E5:12 C5:24",
		"F4:12 A4:12 C5:12 F5:12 E5:12 C5:12 A4:24",
		"E4:12 G4:12 C5:12 E5:12 D5:12 C5:12 G4:24",
		"B4:12 D5:12 G5:24 F5:12 D5:12 B4:24",
	)},
	{Voice: pad, Notes: audio.Notes("C5:96", "A4:96", "G4:96", "B4:96")},
	{Voice: bass, Notes: audio.Notes(
		"A2:12 A3:12 A2:12 A3:12 A2:12 A3:12 A2:12 A3:12",
		"F2:12 F3:12 F2:12 F3:12 F2:12 F3:12 F2:12 F3:12",
		"C3:12 C4:12 C3:12 C4:12 C3:12 C4:12 C3:12 C4:12",
		"G2:12 G3:12 G2:12 G3:12 G2:12 G3:12 G2:12 G3:12",
	)},
	{Voice: drums, Notes: drumBars(4)},
}}

// Voices for the tune.
var (
	lead  = audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(8), Sustain: 0.5, Release: audio.Ticks(3), Volume: 0.7}
	pad   = audio.Voice{Wave: audio.Pulse12, Attack: audio.Ticks(6), Release: audio.Ticks(10), Volume: 0.3}
	bass  = audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(6), Sustain: 0.6, Release: audio.Ticks(2)}
	drums = audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(4), Release: audio.Ticks(1), Volume: 0.4}
)

// drumBars is n bars of kick and snare on the beats with hats between, as
// noise at low, middle and high rates.
func drumBars(n int) []audio.Note {
	var out []audio.Note
	for range n {
		for _, hz := range []float64{300, 12000, 3000, 12000, 300, 12000, 3000, 12000} {
			out = append(out, audio.Note{Hz: hz, Dur: audio.Ticks(12)})
		}
	}
	return out
}

// Sounds are the demo's sounds, for the sound test.
var Sounds = audio.Bank{
	Tunes:   []*audio.Tune{DashTune},
	Effects: []*audio.Effect{Step, Pickup, Bump, GameOver},
}
