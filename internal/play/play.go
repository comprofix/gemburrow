// Package play is the play scene: it runs the world, draws it, and turns its
// events into sounds. It draws for either look: sprites in the pixel look, and
// shapes in the smooth one.
package play

import (
	"fmt"
	"image"

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

// Fire is the game's own input action (DESIGN §4).
const Fire = input.NumActions

// Bindings are the keys for the game's own actions.
var Bindings = []input.Binding{{Action: Fire, Keys: []ebiten.Key{ebiten.KeySpace}}}

// Animation and effects.
const (
	jewelTicks  = 20 // ticks between the jewel's frames
	goblinTicks = 12 // ticks between the goblin's frames
	stepPixels  = 8  // pixels the dwarf moves between walking frames
	flashTicks  = 6  // the caught dwarf flashes at this rate
	digTicks    = 8  // ticks between the pick-axe's swings
	shakeTicks  = 14
	shakePixels = 3
	// A wobbling sack shifts this far, this often, and gold flickers for its
	// last goldWarnTicks.
	wobblePixels  = 1
	wobbleTicks   = 4
	goldWarnTicks = config.TPS
	// shotReadyX places the HUD's mark that the shot is ready, between the
	// level and the lives.
	shotReadyX = 416
)

// Play is a game in progress.
type Play struct {
	world  *world.World
	level  int
	ticks  int64
	clear  int // ticks left of the level clear banner
	caught int // ticks left of the pause after being caught
	dead   int // ticks since being caught
}

// New starts a game at a skill.
func New(ctx *scene.Context, skill config.Skill) scene.Scene {
	ctx.Session = session.New(skill, config.StartLives)
	p := &Play{level: max(ctx.StartLevel, 1)}
	p.world = loadLevel(ctx, p.level)
	ctx.Audio.PlayTune(DiggingTune)
	return p
}

// loadLevel starts level n. The layouts are checked by the world's tests, so
// a bad one is a programming error.
func loadLevel(ctx *scene.Context, n int) *world.World {
	l, err := world.Parse(config.LevelFor(n))
	if err != nil {
		panic(err)
	}
	return world.New(ctx.Session, l, config.TuningFor(ctx.Session.Skill, n), ctx.RNG)
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
	switch {
	case p.clear > 0:
		p.clear--
		if p.clear == 0 {
			if p.world.Chase > 0 {
				ctx.Audio.PlayTune(DiggingTune)
			}
			p.level++
			p.world = loadLevel(ctx, p.level)
		}
		return nil
	case p.caught > 0:
		p.caught--
		p.dead++
		if p.caught == 0 {
			if p.world.Over {
				s := ctx.Session
				ctx.Switch(scene.AfterGame(ctx, s.Skill, s.Score))
			} else {
				p.world.Respawn()
				ctx.Audio.PlayTune(DiggingTune)
			}
		}
		return nil
	}
	for _, e := range p.world.Update(heldDir(in), in.Held(Fire)) {
		switch e {
		case world.Dug, world.SackPushed:
			ctx.Audio.PlayEffect(Dig)
		case world.JewelTaken:
			ctx.Audio.PlayEffect(Jewel)
		case world.JewelRun:
			ctx.Audio.PlayEffect(JewelRun)
		case world.LevelCleared:
			ctx.Audio.PlayEffect(LevelClear)
			p.clear = config.LevelClearTicks
		case world.ShotFired:
			ctx.Audio.PlayEffect(Fired)
		case world.GoblinShot, world.GoblinCrushed:
			ctx.Audio.PlayEffect(GoblinDown)
		case world.SackWobbling:
			ctx.Audio.PlayEffect(SackShake)
		case world.SackLanded:
			ctx.Audio.PlayEffect(SackThud)
		case world.SackBurst:
			ctx.Audio.PlayEffect(SackBurst)
		case world.GoldTaken:
			ctx.Audio.PlayEffect(GoldTaken)
		case world.MushroomAppeared:
			ctx.Audio.PlayEffect(MushroomUp)
		case world.MushroomTaken:
			ctx.Audio.PlayEffect(MushroomTaken)
			ctx.Audio.PlayTune(ChaseTune)
		case world.GoblinEaten:
			ctx.Audio.PlayEffect(GoblinEaten)
		case world.ChaseOver:
			ctx.Audio.PlayTune(DiggingTune)
		case world.ExtraLife:
			ctx.Audio.PlayEffect(LifeGained)
		case world.DwarfCaught:
			ctx.Audio.PlayEffect(Caught)
			ctx.Audio.PlayTune(FuneralTune)
			ctx.Shake.Start(shakeTicks, shakePixels)
			p.dead = 0
			p.caught = config.CaughtTicks
			if p.world.Over {
				p.caught = config.GameOverTicks
			}
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
	if ctx.Preset.Art == config.ArtSmooth {
		p.drawSmooth(ctx, dst)
		return
	}
	sc, f := ctx.Scheme, ctx.Fonts
	dst.Fill(gfx.RGBA(sc.Backdrop))
	sheet := ctx.Sheet("gemburrow")
	w := p.world

	// The HUD (DESIGN §3).
	ty := (config.HUDHeight - f.Large.LineHeight()) / 2
	f.Large.DrawShadowed(dst, fmt.Sprintf("SCORE %06d", ctx.Session.Score), config.HUDX, ty, sc.Text, sc.Shadow)
	ui.TextCentred(dst, f.Large, fmt.Sprintf("LEVEL %d", p.level), ty, sc.Text, sc.Shadow)
	lives := fmt.Sprintf("LIVES %d", ctx.Session.Lives)
	f.Large.DrawShadowed(dst, lives, ctx.Preset.ScreenWidth-config.HUDX-f.Large.Width(lives), ty, sc.Text, sc.Shadow)
	shot := sheet.Sprite("shot")
	if w.Recharge == 0 {
		shot.Draw(dst, 0, shotReadyX, float64((config.HUDHeight-shot.H)/2), false)
	}

	// The field.
	earth, tunnel, jewel := sheet.Sprite("earth"), sheet.Sprite("tunnel"), sheet.Sprite("jewel")
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

	// The gold, flickering as it runs out, and the sacks, shaking before they
	// fall.
	gold := sheet.Sprite("gold")
	for _, g := range w.Gold {
		if g.Ticks > goldWarnTicks || (p.ticks/flashTicks)%2 == 0 {
			gold.Draw(dst, 0, float64(config.FieldX+g.X*config.CellSize), float64(config.FieldY+g.Y*config.CellSize), false)
		}
	}
	sack := sheet.Sprite("sack")
	for _, s := range w.Sacks {
		sx := config.FieldX + s.X
		if s.State == world.SackWobble {
			sx += wobblePixels * (int(p.ticks/wobbleTicks)%2*2 - 1)
		}
		sack.Draw(dst, 0, float64(sx), float64(config.FieldY+s.Y), false)
	}

	if w.Mushroom {
		sheet.Sprite("mushroom").Draw(dst, 0, float64(config.FieldX+config.MushroomCol*config.CellSize), float64(config.FieldY+config.MushroomRow*config.CellSize), false)
	}

	// The goblins and the shot.
	goblin, troll := sheet.Sprite("goblin"), sheet.Sprite("troll")
	if p.fleeing() {
		goblin, troll = sheet.Sprite("goblin_scared"), sheet.Sprite("troll_scared")
	}
	for i, g := range w.Goblins {
		if p.buried(g) {
			continue
		}
		picture := goblin
		if g.Troll > 0 {
			picture = troll
		}
		picture.Draw(dst, int(p.ticks/goblinTicks)+i, float64(config.FieldX+g.X), float64(config.FieldY+g.Y), g.FacesLeft)
		if debug.Enabled {
			debug.Box(dst, float64(config.FieldX+g.X), float64(config.FieldY+g.Y), config.CellSize, config.CellSize, sc.Highlight)
		}
	}
	if s := w.Shot; s != nil {
		shot.Draw(dst, 0, float64(config.FieldX+s.X-shot.W/2), float64(config.FieldY+s.Y-shot.H/2), false)
	}

	// The dwarf, flashing once caught, then its headstone.
	d := w.Dwarf
	dx, dy := config.FieldX+d.X, config.FieldY+d.Y
	if p.headstone() {
		if !p.headstoneShown() {
			// Flashing, off.
		} else {
			clip := dst.SubImage(image.Rect(dx, dy, dx+config.CellSize, dy+config.CellSize)).(*ebiten.Image)
			sheet.Sprite("headstone").Draw(clip, 0, float64(dx), float64(dy+p.headstoneDrop(config.CellSize)), false)
		}
	} else if !w.Caught || (p.ticks/flashTicks)%2 == 0 {
		if d.Digging {
			sheet.Sprite("dwarf_dig").Draw(dst, int(p.ticks/digTicks), float64(dx), float64(dy), d.FacesLeft)
		} else {
			sheet.Sprite("dwarf").Draw(dst, (d.X+d.Y)/stepPixels, float64(dx), float64(dy), d.FacesLeft)
		}
	}
	if debug.Enabled {
		debug.Box(dst, float64(dx), float64(dy), config.CellSize, config.CellSize, sc.Highlight)
	}

	switch {
	case p.clear > 0:
		banner(ctx, dst, "LEVEL CLEAR")
	case w.Over:
		banner(ctx, dst, "GAME OVER")
	}
}

// headstone reports whether the caught dwarf has given way to its headstone.
func (p *Play) headstone() bool {
	return p.world.Caught && p.dead >= config.CaughtFlashTicks
}

// headstoneShown reports whether the headstone is drawn this tick: it flashes
// for the end of the pause, before the dwarf reappears (DESIGN §5). The last
// life's stands to the end.
func (p *Play) headstoneShown() bool {
	return p.world.Over || p.caught > config.HeadstoneFlashTicks || (p.ticks/flashTicks)%2 == 0
}

// headstoneDrop is how far the rising headstone still is below its place, in a
// cell of the given size.
func (p *Play) headstoneDrop(cell int) int {
	left := config.HeadstoneRiseTicks - (p.dead - config.CaughtFlashTicks)
	return cell * max(left, 0) / config.HeadstoneRiseTicks
}

// buried reports whether a goblin is out of sight, gone where the headstone
// stands.
func (p *Play) buried(g world.Goblin) bool {
	d := p.world.Dwarf
	return p.headstone() && g.X-d.X < config.CellSize && d.X-g.X < config.CellSize &&
		g.Y-d.Y < config.CellSize && d.Y-g.Y < config.CellSize
}

// fleeing reports whether goblins are drawn afraid: all through a chase, flashing
// at its start and end (DESIGN §6).
func (p *Play) fleeing() bool {
	w := p.world
	if w.Chase == 0 {
		return false
	}
	if w.Chase > config.ChaseWarnTicks && w.Chase < w.Tuning.ChaseTicks-config.ChaseWarnTicks {
		return true
	}
	return (p.ticks/flashTicks)%2 == 0
}

// banner dims the field under a message.
func banner(ctx *scene.Context, dst *ebiten.Image, msg string) {
	sc, f := ctx.Scheme, ctx.Fonts
	scene.Dim(dst, sc.Shadow, 0.5)
	ui.TextCentred(dst, f.Large, msg, (ctx.Preset.ScreenHeight-f.Large.LineHeight())/2, sc.Title, sc.Shadow)
}
