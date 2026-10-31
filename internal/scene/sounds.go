package scene

import (
	"github.com/comprofix/gameskeleton/internal/audio"
	"github.com/comprofix/gameskeleton/internal/ui"
)

// The shared scenes' sounds. In Chip4, channel 0 carries the tune's lead,
// 1 its harmony, 2 its bass (triangle) and 3 its drums (noise); menu
// sounds take the harmony channel so the melody carries on.
var (
	// MenuMove is the cursor moving or a value changing.
	MenuMove = &audio.Effect{Name: "menu move", Channel: 1, Priority: 1, Part: audio.Part{
		Voice: blip,
		Notes: []audio.Note{audio.N("E6", 3)},
	}}
	// MenuConfirm is a menu item chosen.
	MenuConfirm = &audio.Effect{Name: "menu confirm", Channel: 1, Priority: 2, Part: audio.Part{
		Voice: blip,
		Notes: []audio.Note{audio.N("C6", 3), audio.N("G6", 6)},
	}}

	// TallyTick is a point counting up on the tally.
	TallyTick = &audio.Effect{Name: "tally tick", Channel: 1, Priority: 1, Part: audio.Part{
		Voice: blip,
		Notes: []audio.Note{audio.N("A6", 2)},
	}}

	// TitleTune loops on the title and menus, unless the game has its own
	// (Context.TitleTune): four bars of C, Am, F, G.
	TitleTune = &audio.Tune{Name: "title", Loop: true, Parts: []audio.Part{
		{Voice: lead, Notes: audio.Notes(
			"E5:12 G5:12 C6:24 B5:12 G5:12 E5:24",
			"A5:12 C6:12 E6:24 D6:12 C6:12 A5:24",
			"F5:12 A5:12 C6:24 A5:12 F5:12 G5:12 A5:12",
			"B5:24 D6:12 B5:12 G5:24 -:24",
		)},
		{Voice: pad, Notes: audio.Notes(
			"C5:48 E5:48",
			"C5:48 E5:48",
			"A4:48 C5:48",
			"B4:48 D5:48",
		)},
		{Voice: bass, Notes: audio.Notes(
			"C3:24 G3:24 C3:24 G3:24",
			"A2:24 E3:24 A2:24 E3:24",
			"F2:24 C3:24 F2:24 C3:24",
			"G2:24 D3:24 G2:24 D3:24",
		)},
		{Voice: drums, Notes: drumBar(4)},
	}}
)

// Sounds are the shared scenes' sounds, for the sound test.
var Sounds = audio.Bank{
	Tunes:   []*audio.Tune{TitleTune},
	Effects: []*audio.Effect{MenuMove, MenuConfirm, TallyTick},
}

// Voices.
var (
	blip  = audio.Voice{Wave: audio.Pulse50, Decay: audio.Ticks(6), Sustain: 0.3, Release: audio.Ticks(2), Volume: 0.8}
	lead  = audio.Voice{Wave: audio.Pulse25, Decay: audio.Ticks(10), Sustain: 0.6, Release: audio.Ticks(3), Volume: 0.8}
	pad   = audio.Voice{Wave: audio.Pulse12, Attack: audio.Ticks(3), Release: audio.Ticks(6), Volume: 0.35}
	bass  = audio.Voice{Wave: audio.Triangle, Decay: audio.Ticks(12), Sustain: 0.7, Release: audio.Ticks(2)}
	drums = audio.Voice{Wave: audio.Noise, Decay: audio.Ticks(5), Release: audio.Ticks(1), Volume: 0.45}
)

// drumBar is n bars of kick and snare on the beats with hats between, as
// noise at low, middle and high rates.
func drumBar(n int) []audio.Note {
	const (
		kick  = 300
		snare = 3000
		hat   = 12000
	)
	var out []audio.Note
	for range n {
		for _, hz := range []float64{kick, hat, snare, hat, kick, hat, snare, hat} {
			out = append(out, audio.Note{Hz: hz, Dur: audio.Ticks(12)})
		}
	}
	return out
}

// menuSounds gives m the menu sounds.
func menuSounds(ctx *Context, m *ui.Menu) {
	m.OnMove = func() { ctx.Audio.PlayEffect(MenuMove) }
	m.OnConfirm = func() { ctx.Audio.PlayEffect(MenuConfirm) }
}

// playTitleTune starts the game's title tune, or the skeleton's if it has
// none, unless it is already playing.
func playTitleTune(ctx *Context) {
	t := ctx.TitleTune
	if t == nil {
		t = TitleTune
	}
	if ctx.Audio.Tune() != t {
		ctx.Audio.PlayTune(t)
	}
}

// playScoresTune starts the game's high score tune, if it has one, unless
// it is already playing.
func playScoresTune(ctx *Context) {
	if t := ctx.ScoresTune; t != nil && ctx.Audio.Tune() != t {
		ctx.Audio.PlayTune(t)
	}
}
