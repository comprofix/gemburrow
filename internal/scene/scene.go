// Package scene runs the game's screens: the scene manager with fades and
// screen shake, and the shared scenes.
package scene

import (
	"fmt"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/audio"
	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/gfx"
	"github.com/comprofix/gameskeleton/internal/input"
	"github.com/comprofix/gameskeleton/internal/score"
	"github.com/comprofix/gameskeleton/internal/session"
)

// Scene is one screen of the game.
type Scene interface {
	// Update advances the scene by one tick.
	Update(ctx *Context) error
	// Draw renders onto the internal screen. It must not change state.
	Draw(ctx *Context, dst *ebiten.Image)
}

// Native is a scene that works at the screen's own size in either look.
// Every shared scene is one. In a game with two looks, any other scene
// lives in the pixel look: while the smooth look is shown, it is made,
// updated and drawn with the pixel look's preset and fonts, on a screen of
// its size, then enlarged with sharp pixels. So a game can redraw its
// scenes for the smooth look one at a time. The pointer is still in the
// shown screen's pixels, so such a scene divides it by the scale between
// the looks.
type Native interface{ Native() }

// inPixelLook reports whether s lives in the pixel look now: it is not
// Native and the smooth look of two is shown.
func (c *Context) inPixelLook(s Scene) bool {
	_, ok := s.(Native)
	return !ok && c.smoothOfTwo()
}

// smoothOfTwo reports whether the game has two looks and the smooth one is
// shown.
func (c *Context) smoothOfTwo() bool {
	return c.Looks.Pixel() != nil && c.Preset.Art == config.ArtSmooth
}

// asPixelLook runs f with the context in the pixel look, then puts it back.
// The context itself is changed, not a copy, so what f asks of it (Switch,
// Quit and so on) is kept.
func (c *Context) asPixelLook(f func()) {
	p, fonts := c.Preset, c.Fonts
	c.Preset, c.Fonts = *c.Looks.Pixel(), c.PixelFonts
	defer func() { c.Preset, c.Fonts = p, fonts }()
	f()
}

// makeGame makes one of the game's own scenes through f (NewPlay or
// NewDemo) in the look it lives in: the pixel look while the smooth look of
// two is shown, unless NativePlay says it is redrawn for the smooth look.
// It panics if the scene made does not match NativePlay.
func (c *Context) makeGame(f func(*Context) Scene) Scene {
	if c.NativePlay || !c.smoothOfTwo() {
		return checkNative(c, f(c))
	}
	var s Scene
	c.asPixelLook(func() { s = f(c) })
	return checkNative(c, s)
}

// checkNative panics if s, made by the game, is Native and NativePlay
// says otherwise, or the other way round. Only a game with two looks is
// checked.
func checkNative(c *Context, s Scene) Scene {
	if _, ok := s.(Native); c.Looks.Two() && ok != c.NativePlay {
		panic(fmt.Sprintf("scene: %T is Native: %v, but NativePlay is %v", s, ok, c.NativePlay))
	}
	return s
}

// updateScene updates s in the look it lives in.
func updateScene(ctx *Context, s Scene) error {
	if !ctx.inPixelLook(s) {
		return s.Update(ctx)
	}
	var err error
	ctx.asPixelLook(func() { err = s.Update(ctx) })
	return err
}

// drawScene draws s onto dst, the screen, enlarging it from the pixel look
// if it lives there.
func drawScene(ctx *Context, s Scene, dst *ebiten.Image) {
	if !ctx.inPixelLook(s) {
		s.Draw(ctx, dst)
		return
	}
	px := ctx.Looks.Pixel()
	if ctx.pixel == nil || ctx.pixel.Bounds().Dx() != px.ScreenWidth || ctx.pixel.Bounds().Dy() != px.ScreenHeight {
		if ctx.pixel != nil {
			ctx.pixel.Deallocate()
		}
		ctx.pixel = ebiten.NewImage(px.ScreenWidth, px.ScreenHeight)
	}
	ctx.pixel.Clear()
	ctx.asPixelLook(func() { s.Draw(ctx, ctx.pixel) })
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(dst.Bounds().Dx())/float64(px.ScreenWidth), float64(dst.Bounds().Dy())/float64(px.ScreenHeight))
	op.Filter = ebiten.FilterNearest
	dst.DrawImage(ctx.pixel, op)
}

// NewRNG is the random number generator for a seed. The game and tests
// both use it, so a seed gives the same game in each.
func NewRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
}

// Context is the shared state handed to scenes. Games add their own fields.
type Context struct {
	Input *input.Input
	RNG   *rand.Rand
	// Preset is the look being shown: its screen size, scale modes and art.
	// The game sets it; scenes read the screen's size here, never from
	// config.Project.
	Preset config.Preset
	// Looks are the game's looks, one or two; Settings.Look picks the one
	// shown.
	Looks config.Looks
	// PixelFonts are the pixel look's fonts in a game with two looks, for
	// the scenes drawn in it while the smooth look is shown (see Native).
	PixelFonts gfx.Fonts

	Fonts    gfx.Fonts
	Scheme   gfx.Scheme
	Sheets   map[string]*gfx.Sheet
	Settings *config.Settings
	Shake    Shaker
	Tick     int64        // ticks since the game started
	Audio    *audio.Synth // nil in tests: every call is then ignored
	Sounds   audio.Bank   // every tune and effect, for the sound test
	// The game's own music for the shared scenes: the title and menus, and
	// the high score screens. A nil title tune plays the skeleton's
	// TitleTune; a nil high score tune leaves the music as it is.
	TitleTune, ScoresTune *audio.Tune
	Session               *session.Session // the game in progress, if any
	Scores                *score.Tables

	// NewPlay starts a game at the given skill. The game package sets it,
	// so the shared scenes don't depend on the game's own packages.
	NewPlay func(ctx *Context, skill config.Skill) Scene
	// NewDemo, if set, starts the game playing itself, which the title
	// shows behind its logo.
	NewDemo func(ctx *Context) Scene
	// NativePlay says the scenes NewPlay and NewDemo make are Native, drawn
	// for the smooth look as well as the pixel one. Until then a game with
	// two looks leaves it false and they are made in the pixel look.
	NativePlay bool

	// StartLevel is the level new games start on; 0 or 1 is the first.
	// The -level flag sets it, for testing; a game with levels starts
	// its play scene there. A game started later is a test: it goes on no
	// high score table.
	StartLevel int

	pixel           *ebiten.Image // the pixel look's screen, for scenes enlarged in the smooth look; a render cache
	next            Scene
	cut             bool
	quit            bool
	settingsChanged bool
	scoresChanged   bool
}

// Testing reports whether games are started for testing, part way in:
// they go on no high score table.
func (c *Context) Testing() bool { return c.StartLevel > 1 }

// Switch asks the manager to fade to s at the end of this tick.
func (c *Context) Switch(s Scene) { c.next, c.cut = s, false }

// Cut asks the manager to change to s at the end of this tick without a
// fade, e.g. to open or close the pause menu.
func (c *Context) Cut(s Scene) { c.next, c.cut = s, true }

// Quit asks the game to save and close.
func (c *Context) Quit() { c.quit = true }

// Quitting reports whether a scene asked to quit.
func (c *Context) Quitting() bool { return c.quit }

// SettingsChanged records that a scene changed the settings, so the game
// applies and saves them.
func (c *Context) SettingsChanged() { c.settingsChanged = true }

// TakeSettingsChanged reports whether the settings changed since the last
// call, and clears the flag.
func (c *Context) TakeSettingsChanged() bool {
	ch := c.settingsChanged
	c.settingsChanged = false
	return ch
}

// ScoresChanged records that a score was added, so the game saves the
// tables.
func (c *Context) ScoresChanged() { c.scoresChanged = true }

// TakeScoresChanged reports whether the scores changed since the last call,
// and clears the flag.
func (c *Context) TakeScoresChanged() bool {
	ch := c.scoresChanged
	c.scoresChanged = false
	return ch
}

// Sheet returns the named sprite sheet. It panics if it is missing, since
// sheets are fixed at build time.
func (c *Context) Sheet(name string) *gfx.Sheet {
	sh, ok := c.Sheets[name]
	if !ok {
		panic("scene: no sprite sheet named " + name)
	}
	return sh
}
