// Package play is the play scene: it runs the world, draws it, and turns its
// events into sounds. It is pixel art, enlarged in the smooth look until it
// is redrawn natively.
package play

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gemburrow/internal/config"
	"github.com/comprofix/gemburrow/internal/debug"
	"github.com/comprofix/gemburrow/internal/gfx"
	"github.com/comprofix/gemburrow/internal/input"
	"github.com/comprofix/gemburrow/internal/scene"
	"github.com/comprofix/gemburrow/internal/session"
	"github.com/comprofix/gemburrow/internal/ui"
	"github.com/comprofix/gemburrow/internal/world"
)

// Animation.
const (
	jewelTicks = 20 // ticks between the jewel's frames
	stepPixels = 8  // pixels the dwarf moves between walking frames
)

// Play is a game in progress.
type Play struct {
	world *world.World
	level int
	ticks int64
	clear int // ticks left of the level clear banner
}

// New starts a game at a skill.
func New(ctx *scene.Context, skill config.Skill) scene.Scene {
	ctx.Session = session.New(skill, config.StartLives)
	p := &Play{level: max(ctx.StartLevel, 1)}
	p.world = loadLevel(ctx.Session, p.level)
	return p
}

// loadLevel starts level n. The layouts are checked by the world's tests, so
// a bad one is a programming error.
func loadLevel(s *session.Session, n int) *world.World {
	l, err := world.Parse(config.LevelFor(n))
	if err != nil {
		panic(err)
	}
	return world.New(s, l)
}

// World is the game's state, for tests.
func (p *Play) World() *world.World { return p.world }

// Level is the current level, counting from 1.
func (p *Play) Level() int { return p.level }

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
		ctx.Switch(New(ctx, ctx.Session.Skill))
		return nil
	}
	p.ticks++
	if p.clear > 0 {
		p.clear--
		if p.clear == 0 {
			p.level++
			p.world = loadLevel(ctx.Session, p.level)
		}
		return nil
	}
	for _, e := range p.world.Update(heldDir(in)) {
		switch e {
		case world.Dug:
			ctx.Audio.PlayEffect(Dig)
		case world.JewelTaken:
			ctx.Audio.PlayEffect(Jewel)
		case world.LevelCleared:
			ctx.Audio.PlayEffect(LevelClear)
			p.clear = config.LevelClearTicks
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
	dst.Fill(gfx.RGBA(sc.Backdrop))

	// The HUD (DESIGN §3).
	ty := (config.HUDHeight - f.Large.LineHeight()) / 2
	f.Large.DrawShadowed(dst, fmt.Sprintf("SCORE %06d", ctx.Session.Score), config.FieldX, ty, sc.Text, sc.Shadow)
	ui.TextCentred(dst, f.Large, fmt.Sprintf("LEVEL %d", p.level), ty, sc.Text, sc.Shadow)
	lives := fmt.Sprintf("LIVES %d", ctx.Session.Lives)
	f.Large.DrawShadowed(dst, lives, config.FieldX+config.FieldCols*config.CellSize-f.Large.Width(lives), ty, sc.Text, sc.Shadow)

	// The field.
	sheet := ctx.Sheet("gemburrow")
	earth, tunnel, jewel := sheet.Sprite("earth"), sheet.Sprite("tunnel"), sheet.Sprite("jewel")
	w := p.world
	for y := range config.FieldRows {
		for x := range config.FieldCols {
			sx, sy := float64(config.FieldX+x*config.CellSize), float64(config.FieldY+y*config.CellSize)
			if w.Earth(x, y) {
				earth.Draw(dst, x+y, sx, sy, false)
			} else {
				tunnel.Draw(dst, 0, sx, sy, false)
			}
			if w.Jewel(x, y) {
				jewel.Draw(dst, int(p.ticks/jewelTicks)+x, sx, sy, false)
			}
		}
	}

	// The dwarf.
	d := w.Dwarf
	dx, dy := config.FieldX+d.X, config.FieldY+d.Y
	sheet.Sprite("dwarf").Draw(dst, (d.X+d.Y)/stepPixels, float64(dx), float64(dy), d.FacesLeft)
	if debug.Enabled {
		debug.Box(dst, float64(dx), float64(dy), config.CellSize, config.CellSize, sc.Highlight)
	}

	if p.clear > 0 {
		scene.Dim(dst, sc.Shadow, 0.5)
		ui.TextCentred(dst, f.Large, "LEVEL CLEAR", (ctx.Preset.ScreenHeight-f.Large.LineHeight())/2, sc.Title, sc.Shadow)
	}
}
