package scene

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/input"
	"github.com/comprofix/gameskeleton/internal/ui"
)

// SoundTest plays each tune and effect in Context.Sounds on demand, to
// check how they sound in the project's audio style.
type SoundTest struct {
	back func() Scene
	menu ui.Menu
}

// NewSoundTest returns the sound test; leaving it switches to back().
func NewSoundTest(back func() Scene) *SoundTest { return &SoundTest{back: back} }

// Update implements Scene.
func (s *SoundTest) Update(ctx *Context) error {
	if s.menu.Items == nil {
		for _, t := range ctx.Sounds.Tunes {
			s.menu.Items = append(s.menu.Items, ui.Item{Label: "Tune: " + t.Name, Confirm: func() { ctx.Audio.PlayTune(t) }})
		}
		for _, e := range ctx.Sounds.Effects {
			s.menu.Items = append(s.menu.Items, ui.Item{Label: e.Name, Confirm: func() { ctx.Audio.PlayEffect(e) }})
		}
		s.menu.Items = append(s.menu.Items,
			ui.Item{Label: "Stop", Confirm: ctx.Audio.Stop},
			ui.Item{Label: "Back", Confirm: func() { s.leave(ctx) }},
		)
		// Only the move sound: a confirm sound would cut off the effect
		// being tested.
		s.menu.OnMove = func() { ctx.Audio.PlayEffect(MenuMove) }
		s.menu.Rows = menuRows(ctx) // the list is long: it scrolls
	}
	if ctx.Input.JustPressed(input.Back) {
		s.leave(ctx)
		return nil
	}
	updateMenu(ctx, &s.menu, menuTop)
	return nil
}

func (s *SoundTest) leave(ctx *Context) {
	ctx.Audio.Stop()
	ctx.Switch(s.back())
}

// Draw implements Scene.
func (s *SoundTest) Draw(ctx *Context, dst *ebiten.Image) {
	drawMenuScreen(ctx, dst, "SOUND TEST", &s.menu, "Enter: play   Esc: back")
}
