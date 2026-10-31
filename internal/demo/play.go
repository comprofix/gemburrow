// Package demo is Dot Dash, the skeleton's demo game. Nothing
// outside it imports it except the game package, which hands NewPlay to the
// scenes; tools/newgame removes it. Its rules are in the world package.
package demo

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/debug"
	"github.com/comprofix/gameskeleton/internal/demo/world"
	"github.com/comprofix/gameskeleton/internal/gfx"
	"github.com/comprofix/gameskeleton/internal/input"
	"github.com/comprofix/gameskeleton/internal/scene"
	"github.com/comprofix/gameskeleton/internal/session"
	"github.com/comprofix/gameskeleton/internal/ui"
)

// Layout, in screen pixels. The board takes whatever room is left, so it
// fits both presets.
const (
	Cell   = 16 // one board cell; the player and hazard sprites fill it
	HUD    = 24 // the score line across the top
	Margin = 8  // around the board
)

// Effects and animation, in ticks.
const (
	shakeTicks    = 12
	shakePx       = 3
	slideTicks    = 4 // a mover slides to its new cell over this many ticks
	flashTicks    = 4 // the hurt player flashes at this rate while safe
	gameOverTicks = 2 * config.TPS
	bobTicks      = 10
	dotTicks      = 8
	hazardTicks   = 15
)

// Board is the size of the board on preset p's screen, in cells.
func Board(p config.Preset) (cols, rows int) {
	return (p.ScreenWidth - 2*Margin) / Cell, (p.ScreenHeight - HUD - Margin) / Cell
}

// Play is Dot Dash in progress.
type Play struct {
	world *world.World
	ticks int64
}

// NewPlay starts a game at a skill.
func NewPlay(ctx *scene.Context, skill config.Skill) scene.Scene {
	ctx.Audio.PlayTune(DashTune)
	ctx.Session = session.New(skill, 0)
	cols, rows := Board(ctx.Preset)
	return &Play{world: world.New(ctx.Session, cols, rows, world.ForSkill(skill.ID), ctx.RNG)}
}

// World is the game's state, for tests.
func (p *Play) World() *world.World { return p.world }

// Update implements scene.Scene.
func (p *Play) Update(ctx *scene.Context) error {
	in := ctx.Input
	switch {
	case in.JustPressed(input.Pause):
		ctx.Cut(scene.NewPause(p))
		return nil
	case in.JustPressed(input.Menu):
		ctx.Switch(scene.NewMainMenu())
		return nil
	case in.JustPressed(input.Restart):
		ctx.Switch(NewPlay(ctx, ctx.Session.Skill))
		return nil
	}
	p.ticks++
	for _, e := range p.world.Update(heldDir(in)) {
		switch e {
		case world.Stepped:
			ctx.Audio.PlayEffect(Step)
		case world.Picked:
			ctx.Audio.PlayEffect(Pickup)
		case world.Bumped:
			ctx.Audio.PlayEffect(Bump)
			ctx.Shake.Start(shakeTicks, shakePx)
		case world.TimeUp:
			ctx.Audio.StopTune()
			ctx.Audio.PlayEffect(GameOver)
			ctx.Cut(newEnded(p, "TIME UP"))
		case world.OutOfHearts:
			ctx.Audio.StopTune()
			ctx.Audio.PlayEffect(GameOver)
			ctx.Cut(newEnded(p, "GAME OVER"))
		}
	}
	return nil
}

// heldDir is the direction being pushed: the first held of up, down, left
// and right.
func heldDir(in *input.Input) world.Dir {
	switch {
	case in.Held(input.Up):
		return world.Up
	case in.Held(input.Down):
		return world.Down
	case in.Held(input.Left):
		return world.Left
	case in.Held(input.Right):
		return world.Right
	}
	return world.None
}

// Draw implements scene.Scene.
func (p *Play) Draw(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	w := p.world
	sw := ctx.Preset.ScreenWidth
	dst.Fill(gfx.RGBA(sc.Backdrop))

	// The score line: score, time left, hearts.
	ty := (HUD - f.Small.LineHeight()) / 2
	f.Small.DrawShadowed(dst, fmt.Sprintf("SCORE %06d", ctx.Session.Score), Margin, ty, sc.Text, sc.Shadow)
	secs := (w.TicksLeft + config.TPS - 1) / config.TPS
	timeC := sc.Text
	if secs <= 5 {
		timeC = sc.Highlight
	}
	ui.TextCentred(dst, f.Small, fmt.Sprintf("TIME %2d", secs), ty, timeC, sc.Shadow)
	sheet := ctx.Sheet("demo")
	heart := sheet.Sprite("heart")
	for i := range ctx.Session.Lives {
		heart.Draw(dst, 0, float64(sw-Margin-(i+1)*(heart.W+2)), float64((HUD-heart.H)/2), false)
	}

	// The board.
	ox, oy := boardOrigin(ctx.Preset, w)
	ui.Panel(dst, image.Rect(ox-2, oy-2, ox+w.Cols*Cell+2, oy+w.Rows*Cell+2), sc.Panel, sc.Border)
	dot := sheet.Sprite("dot")
	for i, d := range w.Dots {
		x, y := float64(ox+d.X*Cell+(Cell-dot.W)/2), float64(oy+d.Y*Cell+(Cell-dot.H)/2)
		dot.Draw(dst, int(p.ticks/dotTicks)+i, x, y, false)
	}
	hazard := sheet.Sprite("hazard")
	for i, h := range w.Hazards {
		x, y := slide(h.Mover, ox, oy)
		hazard.Draw(dst, int(p.ticks/hazardTicks)+i, x, y, false)
	}
	player := sheet.Sprite("player")
	if w.SafeTicks > 0 && (w.SafeTicks/flashTicks)%2 == 1 {
		player = sheet.Sprite("player_hurt")
	}
	x, y := slide(w.Player, ox, oy)
	player.Draw(dst, int(p.ticks/bobTicks), x, y, w.FacesLeft)

	// Debug builds outline the cells the rules see, which can differ from
	// where things are drawn while they slide.
	if debug.Enabled {
		box := func(c world.Point) {
			debug.Box(dst, float64(ox+c.X*Cell), float64(oy+c.Y*Cell), Cell, Cell, sc.Highlight)
		}
		box(w.Player.Pos)
		for _, h := range w.Hazards {
			box(h.Pos)
		}
		for _, d := range w.Dots {
			box(d)
		}
	}
}

// boardOrigin is the screen position of the board's top-left cell: centred
// below the score line on preset p's screen.
func boardOrigin(p config.Preset, w *world.World) (x, y int) {
	return (p.ScreenWidth - w.Cols*Cell) / 2, HUD + (p.ScreenHeight-HUD-w.Rows*Cell)/2
}

// slide is where to draw a mover: part way from its last cell to its new
// one, just after it moves.
func slide(m world.Mover, ox, oy int) (float64, float64) {
	t := min(1, float64(m.Since)/slideTicks)
	x := float64(m.Prev.X) + float64(m.Pos.X-m.Prev.X)*t
	y := float64(m.Prev.Y) + float64(m.Pos.Y-m.Prev.Y)*t
	return float64(ox) + x*Cell, float64(oy) + y*Cell
}

// Ended holds the finished board on screen with a message, then moves
// on to the tally. Confirm moves on sooner.
type Ended struct {
	play  *Play
	msg   string
	ticks int
}

func newEnded(p *Play, msg string) *Ended { return &Ended{play: p, msg: msg} }

// Update implements scene.Scene.
func (g *Ended) Update(ctx *scene.Context) error {
	g.ticks++
	if g.ticks >= gameOverTicks || (g.ticks > config.TPS/2 && (ctx.Input.JustPressed(input.Confirm) || ctx.Input.Clicked())) {
		ctx.Switch(tally(ctx, g.play.world))
	}
	return nil
}

// tally adds the end-of-game bonus to the score and counts it all up.
func tally(ctx *scene.Context, w *world.World) scene.Scene {
	s := ctx.Session
	bonus := w.HeartBonus()
	s.AddScore(bonus)
	t := w.Tuning
	return scene.NewTally("WELL DONE", []scene.TallyLine{
		{Label: fmt.Sprintf("Dots    %2d x %d", w.DotsTaken, t.DotPoints), Points: w.DotsTaken * t.DotPoints},
		{Label: fmt.Sprintf("Hearts  %2d x %d", s.Lives, t.HeartPoints), Points: bonus},
	}, func() scene.Scene { return scene.AfterGame(ctx, s.Skill, s.Score) })
}

// Draw implements scene.Scene.
func (g *Ended) Draw(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	g.play.Draw(ctx, dst)
	scene.Dim(dst, sc.Shadow, 0.5)
	ui.TextCentred(dst, f.Large, g.msg, (ctx.Preset.ScreenHeight-f.Large.LineHeight())/2, sc.Title, sc.Shadow)
}
