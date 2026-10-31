package demo

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/audio"
	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/demo/world"
	"github.com/comprofix/gameskeleton/internal/scene"
	"github.com/comprofix/gameskeleton/internal/scene/scenetest"
)

func TestBoardFitsBothPresets(t *testing.T) {
	for _, p := range []config.Preset{config.Modern, config.Classic} {
		cols, rows := Board(p)
		if cols < 2*world.ForSkill("hard").SafeRadius+1 || rows < 8 {
			t.Errorf("%s: board %d×%d is too small", p.Name, cols, rows)
		}
	}
}

// start begins a game at skill on a harness.
func start(t *testing.T, seed uint64, skill config.Skill) (*scenetest.Harness, *Play) {
	h := scenetest.New(t, seed)
	h.Ctx.NewPlay = NewPlay
	play := NewPlay(h.Ctx, skill).(*Play)
	h.Start(play)
	h.Ticks(scene.FadeTicks)
	return h, play
}

// TestAWholeGame plays a game headless, standing still until the clock runs
// out, then goes through game over, the tally and initials to the table.
func TestAWholeGame(t *testing.T) {
	skill := config.Skills[0]
	h, play := start(t, 3, skill)

	h.Ticks(world.ForSkill(skill.ID).Ticks)
	scenetest.Current[*Ended](h)
	h.Ticks(gameOverTicks + 2*scene.FadeTicks)
	tally := scenetest.Current[*scene.Tally](h)
	w := play.World()
	want := w.DotsTaken*w.Tuning.DotPoints + w.HeartBonus()
	if h.Ctx.Session.Score != want {
		t.Errorf("final score %d, want dots plus hearts %d", h.Ctx.Session.Score, want)
	}
	h.Press(ebiten.KeyEnter) // finish counting
	if tally.Total() != h.Ctx.Session.Score {
		t.Errorf("tally total %d, score %d", tally.Total(), h.Ctx.Session.Score)
	}
	h.Press(ebiten.KeyEnter)
	if want == 0 {
		t.Skip("bumped out with no score: nothing to enter") // not with this seed
	}
	scenetest.Current[*scene.HighScoreEntry](h)
	for range config.InitialsLen {
		h.Press(ebiten.KeyEnter)
	}
	scenetest.Current[*scene.HighScoreTable](h)
	if es := h.Ctx.Scores.Table(skill.ID); len(es) != 1 || es[0].Score != want || es[0].Initials != "AAA" {
		t.Errorf("table %v", es)
	}
	if !h.Ctx.TakeScoresChanged() {
		t.Error("the score was not reported for saving")
	}
}

func TestMusicDuringPlay(t *testing.T) {
	h := scenetest.New(t, 3)
	h.Ctx.Audio = audio.NewSynth(config.Chip4)
	h.Ctx.Audio.PlayTune(scene.TitleTune)
	h.Start(NewPlay(h.Ctx, config.Skills[0]))
	if h.Ctx.Audio.Tune() != DashTune {
		t.Fatal("starting a game did not start its tune")
	}
	h.Ticks(scene.FadeTicks + world.ForSkill(config.Skills[0].ID).Ticks)
	if h.Ctx.Audio.Tune() != nil {
		t.Error("the tune is still playing after the game ended")
	}
}

// botKey is the arrow key a simple player would hold: towards the nearest
// dot, across first, paying no attention to hazards.
func botKey(w *world.World) ebiten.Key {
	p := w.Player.Pos
	best, dist := w.Dots[0], -1
	for _, d := range w.Dots {
		if n := abs(d.X-p.X) + abs(d.Y-p.Y); dist < 0 || n < dist {
			best, dist = d, n
		}
	}
	switch {
	case best.X > p.X:
		return ebiten.KeyArrowRight
	case best.X < p.X:
		return ebiten.KeyArrowLeft
	case best.Y > p.Y:
		return ebiten.KeyArrowDown
	}
	return ebiten.KeyArrowUp
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// summary is the game's state as text, for the golden file.
func summary(p *Play, s *scene.Context) string {
	w := p.World()
	var b strings.Builder
	fmt.Fprintf(&b, "board %dx%d  skill %s\n", w.Cols, w.Rows, s.Session.Skill.ID)
	fmt.Fprintf(&b, "over %v  ticks left %d\n", w.Over, w.TicksLeft)
	fmt.Fprintf(&b, "score %d  dots %d  hearts %d\n", s.Session.Score, w.DotsTaken, s.Session.Lives)
	fmt.Fprintf(&b, "player %v\n", w.Player.Pos)
	fmt.Fprintf(&b, "dots %v\n", w.Dots)
	for i, h := range w.Hazards {
		fmt.Fprintf(&b, "hazard %d %v going %v\n", i, h.Pos, h.Dir)
	}
	return b.String()
}

// TestGoldenReplay plays a game from a fixed seed at each skill, with the
// input chosen each tick by botKey, and checks it ends exactly as recorded
// in testdata. A change to
// Dot Dash's rules, tuning or randomness shows up here; if it is intended,
// run the test with -update.
func TestGoldenReplay(t *testing.T) {
	if config.Project.Name != config.Modern.Name {
		t.Skip("the golden files are for the Modern preset's board")
	}
	for _, sk := range config.Skills {
		t.Run(sk.ID, func(t *testing.T) {
			h, play := start(t, 42, sk)
			for !play.World().Over {
				h.Run(scenetest.Hold(1, botKey(play.World())))
			}
			scenetest.Golden(t, "replay_"+sk.ID, summary(play, h.Ctx))
		})
	}
}
