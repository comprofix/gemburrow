package play

import "github.com/comprofix/gemburrow/internal/audio"

// Voices for the game's tunes, all written for Gem Burrow.
var (
	tuneLead = audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(10), Sustain: 0.6, Release: audio.Ticks(3), Volume: 0.7}
	tunePad  = audio.Voice{Wave: audio.Pulse12, Attack: audio.Ticks(3), Release: audio.Ticks(6), Volume: 0.3}
	tuneBass = audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(12), Sustain: 0.7, Release: audio.Ticks(2), Volume: 0.8}
	tuneBeat = audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(5), Release: audio.Ticks(1), Volume: 0.3}
	playLead = audio.Voice{Wave: audio.Pulse50, Decay: audio.Ticks(8), Sustain: 0.4, Release: audio.Ticks(2), Volume: 0.45}
	playBass = audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(8), Sustain: 0.5, Release: audio.Ticks(2), Volume: 0.6}
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
	// TitleTune is for the title and menus.
	TitleTune = &audio.Tune{Name: "title", Loop: true, Parts: []audio.Part{
		{Voice: tuneLead, Notes: audio.Notes(
			"A4:12 C5:12 E5:12 A5:12 G5:12 E5:12 C5:12 D5:12",
			"E5:24 D5:12 C5:12 B4:24 G4:24",
			"F4:12 A4:12 C5:12 F5:12 E5:12 C5:12 A4:12 B4:12",
			"C5:24 B4:12 A4:12 A4:48",
		)},
		{Voice: tunePad, Notes: audio.Notes(
			"A4:48 E5:48",
			"G4:48 D5:48",
			"F4:48 C5:48",
			"E4:48 E4:48",
		)},
		{Voice: tuneBass, Notes: audio.Notes(
			"A2:24 E3:24 A2:24 E3:24",
			"G2:24 D3:24 G2:24 D3:24",
			"F2:24 C3:24 F2:24 C3:24",
			"E2:24 B2:24 A2:48",
		)},
		{Voice: tuneBeat, Notes: beat(4, 12)},
	}}

	// DiggingTune plays under the game. It leaves the harmony channel to the
	// sound effects.
	DiggingTune = &audio.Tune{Name: "digging", Loop: true, Parts: []audio.Part{
		{Voice: playLead, Notes: audio.Notes(
			"E5:10 -:10 G5:10 E5:10 B4:10 -:10 D5:10 E5:10",
			"G5:10 -:10 A5:10 G5:10 E5:10 -:10 D5:20",
			"C5:10 -:10 E5:10 C5:10 G4:10 -:10 B4:10 C5:10",
			"D5:20 B4:10 G4:10 E4:40",
		)},
		{Voice: tunePad, Notes: audio.Notes("-:320")},
		{Voice: playBass, Notes: audio.Notes(
			"E2:20 E3:20 E2:20 E3:20",
			"G2:20 G3:20 G2:20 G3:20",
			"C2:20 C3:20 C2:20 C3:20",
			"B1:20 B2:20 E2:20 E3:20",
		)},
		{Voice: tuneBeat, Notes: beat(4, 10)},
	}}

	// ScoresTune is a short fanfare for the high score screens.
	ScoresTune = &audio.Tune{Name: "high scores", Parts: []audio.Part{
		{Voice: tuneLead, Notes: audio.Notes("C5:10 E5:10 G5:10 C6:20 G5:10 C6:10 E6:40")},
		{Voice: tunePad, Notes: audio.Notes("C4:60 G4:50")},
		{Voice: tuneBass, Notes: audio.Notes("C3:30 G2:30 C3:50")},
	}}
)
