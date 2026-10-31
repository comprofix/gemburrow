package play

import "github.com/comprofix/gemburrow/internal/audio"

// Voices for the game's tunes, all written for Gem Burrow except FuneralTune's
// melody (DESIGN §10, D11).
var (
	tuneLead = audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(10), Sustain: 0.6, Release: audio.Ticks(3), Volume: 0.7}
	tunePad  = audio.Voice{Wave: audio.Pulse12, Attack: audio.Ticks(3), Release: audio.Ticks(6), Volume: 0.3}
	tuneBass = audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(12), Sustain: 0.7, Release: audio.Ticks(2), Volume: 0.8}
	tuneBeat = audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(5), Release: audio.Ticks(1), Volume: 0.3}
	// The theme's plucked voices: each note dies away quickly, for its bounce.
	burrowLead = audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(9), Sustain: 0.12, Release: audio.Ticks(2), Volume: 0.45}
	burrowBass = audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(8), Sustain: 0.2, Release: audio.Ticks(2), Volume: 0.6}
	playLead   = audio.Voice{Wave: audio.Pulse50, Decay: audio.Ticks(8), Sustain: 0.4, Release: audio.Ticks(2), Volume: 0.45}
	playBass   = audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(8), Sustain: 0.5, Release: audio.Ticks(2), Volume: 0.6}
)

// beat is n bars of a low and a high noise, in turn, eight to the bar.
func beat(n, ticks int) []audio.Note {
	const low, high = 400, 9000
	var out []audio.Note
	for range n {
		for i := range 8 {
			hz := float64(high)
			if i%4 == 0 {
				hz = low
			}
			out = append(out, audio.Note{Hz: hz, Dur: audio.Ticks(ticks)})
		}
	}
	return out
}

var (
	// ChaseTune plays under the chase (DESIGN §6). It leaves the harmony channel
	// to the sound effects.
	ChaseTune = &audio.Tune{Name: "chase", Loop: true, Parts: []audio.Part{
		{Voice: playLead, Notes: audio.Notes(
			"A5:6 -:6 E5:6 -:6 A5:6 C6:6 B5:6 G5:6",
			"A5:6 -:6 F5:6 -:6 E5:6 G5:6 E5:12",
		)},
		{Voice: playBass, Notes: audio.Notes(
			"A2:12 A3:12 A2:12 A3:12",
			"F2:12 F3:12 E2:12 E3:12",
		)},
	}}

	// BurrowTune is the game's own theme, for the title, the menus and play: quick,
	// plucked synth notes over a bouncing bass, written for Gem Burrow. It is in
	// two eight-bar sections of 96 ticks a bar, and leaves the harmony channel to
	// the sound effects.
	BurrowTune = &audio.Tune{Name: "burrow", Loop: true, Parts: []audio.Part{
		{Voice: burrowLead, Notes: audio.Notes(
			"G4:12 B4:12 D5:24 C5:12 D5:12 B4:24",
			"A4:12 C5:12 E5:24 D5:12 C5:12 A4:24",
			"F4:12 A4:12 C5:12 F5:12 E5:24 D5:24",
			"G4:12 A4:12 B4:12 D5:12 G5:36 -:12",
			"G4:12 B4:12 D5:24 C5:12 D5:12 B4:24",
			"A4:12 C5:12 E5:24 G5:12 E5:12 C5:24",
			"F5:24 E5:12 D5:12 C5:24 A4:24",
			"B4:12 A4:12 G4:24 G5:12 -:12 G4:12 -:12",

			"E5:12 -:12 E5:12 F#5:12 G5:24 E5:24",
			"C5:12 -:12 C5:12 D5:12 E5:24 C5:24",
			"D5:12 -:12 D5:12 E5:12 F#5:24 A5:24",
			"G5:12 F#5:12 E5:12 D5:12 B4:48",
			"E5:12 -:12 E5:12 F#5:12 G5:24 E5:24",
			"C5:12 -:12 C5:12 D5:12 E5:24 C5:24",
			"A4:12 C5:12 E5:12 A5:12 G5:12 F#5:12 E5:12 D5:12",
			"D5:12 C5:12 B4:12 A4:12 G4:24 -:24",
		)},
		{Voice: tunePad, Notes: audio.Notes("-:1536")},
		{Voice: burrowBass, Notes: audio.Notes(
			"G2:12 G3:12 G2:12 G3:12 G2:12 G3:12 G2:12 G3:12",
			"A2:12 A3:12 A2:12 A3:12 A2:12 A3:12 A2:12 A3:12",
			"F2:12 F3:12 F2:12 F3:12 F2:12 F3:12 F2:12 F3:12",
			"G2:12 G3:12 G2:12 G3:12 G2:12 G3:12 G2:12 G3:12",
			"G2:12 G3:12 G2:12 G3:12 G2:12 G3:12 G2:12 G3:12",
			"A2:12 A3:12 A2:12 A3:12 A2:12 A3:12 A2:12 A3:12",
			"F2:12 F3:12 F2:12 F3:12 F2:12 F3:12 F2:12 F3:12",
			"G2:12 G3:12 G2:12 G3:12 G2:12 G3:12 G2:12 G3:12",

			"E2:12 E3:12 E2:12 E3:12 E2:12 E3:12 E2:12 E3:12",
			"C2:12 C3:12 C2:12 C3:12 C2:12 C3:12 C2:12 C3:12",
			"D2:12 D3:12 D2:12 D3:12 D2:12 D3:12 D2:12 D3:12",
			"G2:12 G3:12 G2:12 G3:12 G2:12 G3:12 G2:12 G3:12",
			"E2:12 E3:12 E2:12 E3:12 E2:12 E3:12 E2:12 E3:12",
			"C2:12 C3:12 C2:12 C3:12 C2:12 C3:12 C2:12 C3:12",
			"A2:12 A3:12 A2:12 A3:12 D2:12 D3:12 D2:12 D3:12",
			"G2:12 G3:12 G2:12 G3:12 G2:12 G3:12 G2:12 G3:12",
		)},
		{Voice: tuneBeat, Notes: beat(16, 12)},
	}}

	// FuneralTune plays once when the dwarf dies: the opening two bars of the
	// funeral march from Chopin's Piano Sonata No. 2, which is in the public
	// domain (DESIGN §5, D11). A beat is 30 ticks, so it lasts 4 seconds.
	FuneralTune = &audio.Tune{Name: "funeral march", Parts: []audio.Part{
		{Voice: tuneLead, Notes: audio.Notes(
			"A#3:30 A#3:22 A#3:8 A#3:30 C#4:22 C4:8",
			"C4:22 A#3:8 A#3:22 A3:8 A#3:60",
		)},
		{Voice: tuneBass, Notes: audio.Notes(
			"A#2:30 F#2:30 A#2:30 F#2:30",
			"A#2:30 F#2:30 A#2:60",
		)},
	}}

	// ScoresTune is a short fanfare for the high score screens.
	ScoresTune = &audio.Tune{Name: "high scores", Parts: []audio.Part{
		{Voice: tuneLead, Notes: audio.Notes("C5:10 E5:10 G5:10 C6:20 G5:10 C6:10 E6:40")},
		{Voice: tunePad, Notes: audio.Notes("C4:60 G4:50")},
		{Voice: tuneBass, Notes: audio.Notes("C3:30 G2:30 C3:50")},
	}}
)
