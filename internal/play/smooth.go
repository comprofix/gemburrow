package play

import (
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gemburrow/internal/config"
	"github.com/comprofix/gemburrow/internal/debug"
	"github.com/comprofix/gemburrow/internal/gfx"
	"github.com/comprofix/gemburrow/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/gemburrow/internal/scene"
	"github.com/comprofix/gemburrow/internal/shapes"
	"github.com/comprofix/gemburrow/internal/ui"
	"github.com/comprofix/gemburrow/internal/world"
)

// artCell is the size of the cell the smooth pictures are laid out in. They
// are drawn at whatever size the screen's cell is, from these coordinates.
const artCell = 32

// Earth's specks: how many a cell has, and their size in art pixels.
const (
	speckCount = 5
	speckSize  = 1.5
	tunnelRoom = 6.0 // the radius of a tunnel's rounded corners, in art pixels
)

// Native implements scene.Native: the play scene is drawn for either look.
func (p *Play) Native() {}

// pen draws a picture laid out in a cell of artCell, at the top-left corner
// (x, y) of a cell of u screen pixels per art pixel, mirrored if asked.
type pen struct {
	dst    *ebiten.Image
	x, y   float64
	u      float64
	mirror bool
}

func (n pen) px(x float64) float64 {
	if n.mirror {
		x = artCell - x
	}
	return n.x + x*n.u
}

func (n pen) py(y float64) float64 { return n.y + y*n.u }

func (n pen) rect(x, y, w, h, radius float64, c int) {
	if n.mirror {
		x = artCell - x - w
	}
	shapes.FillRoundRect(n.dst, shapes.R(n.x+x*n.u, n.y+y*n.u, w*n.u, h*n.u), radius*n.u, c)
}

func (n pen) circle(x, y, r float64, c int) {
	shapes.FillCircle(n.dst, n.px(x), n.py(y), r*n.u, c)
}

func (n pen) poly(c int, pts ...float64) {
	out := make([]float64, len(pts))
	for i := 0; i+1 < len(pts); i += 2 {
		out[i], out[i+1] = n.px(pts[i]), n.py(pts[i+1])
	}
	shapes.FillPoly(n.dst, c, out...)
}

func (n pen) line(x0, y0, x1, y1, width float64, c int) {
	shapes.Line(n.dst, n.px(x0), n.py(y0), n.px(x1), n.py(y1), width*n.u, c)
}

// drawSmooth draws the game for the smooth look. Game coordinates are those
// of the pixel look (DESIGN §3), so everything is scaled by the screens'
// multiple; only the drawing differs.
func (p *Play) drawSmooth(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	dst.Fill(gfx.RGBA(sc.Backdrop))
	w := p.world
	s := ctx.Preset.ScreenWidth / config.Modern.ScreenWidth
	u := float64(s * config.CellSize / artCell)
	at := func(x, y int) pen {
		return pen{dst: dst, x: float64(s * (config.FieldX + x)), y: float64(s * (config.FieldY + y)), u: u}
	}

	// The HUD (DESIGN §3).
	ty := (s*config.HUDHeight - f.Large.LineHeight()) / 2
	left, right := s*config.HUDX, ctx.Preset.ScreenWidth-s*config.HUDX
	f.Large.DrawShadowed(dst, fmt.Sprintf("SCORE %06d", ctx.Session.Score), left, ty, sc.Text, sc.Shadow)
	ui.TextCentred(dst, f.Large, fmt.Sprintf("LEVEL %d", p.level), ty, sc.Text, sc.Shadow)
	lives := fmt.Sprintf("LIVES %d", ctx.Session.Lives)
	f.Large.DrawShadowed(dst, lives, right-f.Large.Width(lives), ty, sc.Text, sc.Shadow)
	if w.Recharge == 0 {
		drawShot(dst, float64(s*shotReadyX+s*config.ShotSize/2), float64(s*config.HUDHeight/2), float64(s*config.ShotSize/2))
	}

	drawField(w, at, u)
	for y := range config.FieldRows {
		for x := range config.FieldCols {
			if w.Jewel(x, y) {
				drawJewel(at(x*config.CellSize, y*config.CellSize), int(p.ticks/jewelTicks)+x)
			}
		}
	}

	// The gold, flickering as it runs out, and the sacks, shaking before they
	// fall.
	for _, g := range w.Gold {
		if g.Ticks > goldWarnTicks || (p.ticks/flashTicks)%2 == 0 {
			drawGold(at(g.X*config.CellSize, g.Y*config.CellSize))
		}
	}
	for _, sack := range w.Sacks {
		x := sack.X
		if sack.State == world.SackWobble {
			x += wobblePixels * (int(p.ticks/wobbleTicks)%2*2 - 1)
		}
		drawSack(at(x, sack.Y))
	}

	if w.Mushroom {
		drawMushroom(at(config.MushroomCol*config.CellSize, config.MushroomRow*config.CellSize), int(p.ticks/jewelTicks))
	}

	// The kobolds and the shot.
	for i, g := range w.Kobolds {
		if p.buried(g) {
			continue
		}
		n := at(g.X, g.Y)
		n.mirror = g.FacesLeft
		if g.Goblin > 0 {
			drawGoblin(n, int(p.ticks/koboldTicks)+i, p.fleeing())
		} else {
			drawKobold(n, int(p.ticks/koboldTicks)+i, p.fleeing())
		}
		if debug.Enabled {
			debug.Box(dst, n.x, n.y, float64(s*config.CellSize), float64(s*config.CellSize), sc.Highlight)
		}
	}
	drawMotes(at(0, 0), p.motes())
	if shot := w.Shot; shot != nil {
		n := at(shot.X, shot.Y)
		drawShot(dst, n.x, n.y, float64(s*config.ShotSize/2))
	}

	// The dwarf, flashing once caught, then its headstone.
	d := w.Dwarf
	n := at(d.X, d.Y)
	n.mirror = d.FacesLeft
	if p.headstone() {
		if p.headstoneShown() {
			cell := s * config.CellSize
			stone := p.headstonePen(n, cell)
			stone.dst = dst.SubImage(image.Rect(int(n.x), int(n.y), int(n.x)+cell, int(n.y)+cell)).(*ebiten.Image)
			drawHeadstone(stone)
		}
	} else if !w.Caught || (p.ticks/flashTicks)%2 == 0 {
		pick := pickOnBack
		if d.Digging {
			pick = pickUp + int(p.ticks/digTicks)%2
		}
		drawDwarf(n, (d.X+d.Y)/stepPixels, pick)
	}
	if debug.Enabled {
		debug.Box(dst, n.x, n.y, float64(s*config.CellSize), float64(s*config.CellSize), sc.Highlight)
	}

	switch {
	case p.clear > 0:
		banner(ctx, dst, "LEVEL CLEAR")
	case w.Over:
		banner(ctx, dst, "GAME OVER")
	}
}

// drawField draws the earth with its specks, and rounded tunnels through it.
func drawField(w *world.World, at func(x, y int) pen, u float64) {
	cell := u * artCell
	corner := at(0, 0)
	shapes.FillRect(corner.dst, shapes.R(corner.x, corner.y, cell*config.FieldCols, cell*config.FieldRows), resurrect64.Rust)
	open := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < config.FieldCols && y < config.FieldRows && !w.Earth(x, y)
	}
	for y := range config.FieldRows {
		for x := range config.FieldCols {
			n := at(x*config.CellSize, y*config.CellSize)
			if !open(x, y) {
				drawSpecks(n, x, y)
				continue
			}
			n.rect(0, 0, artCell, artCell, tunnelRoom, resurrect64.Ink)
			if open(x+1, y) {
				n.rect(artCell/2, 0, artCell, artCell, 0, resurrect64.Ink)
			}
			if open(x, y+1) {
				n.rect(0, artCell/2, artCell, artCell, 0, resurrect64.Ink)
			}
		}
	}
}

// drawSpecks scatters a cell's specks, the same each frame.
func drawSpecks(n pen, x, y int) {
	h := x*31 + y*17
	for i := range speckCount {
		h = (h*67 + 13 + i) % 1021
		c := resurrect64.Maroon
		if h%2 == 0 {
			c = resurrect64.Copper
		}
		sx, sy := float64(h%(artCell-3)), float64((h/7)%(artCell-3))
		shapes.FillRect(n.dst, shapes.R(n.x+sx*n.u, n.y+sy*n.u, speckSize*n.u, speckSize*n.u), c)
	}
}

func drawJewel(n pen, frame int) {
	n.poly(resurrect64.Jade, 9, 8, 23, 8, 28, 14, 16, 27, 4, 14)
	n.poly(resurrect64.Seafoam, 9, 8, 23, 8, 28, 14, 4, 14)
	n.poly(resurrect64.Teal, 4, 14, 12, 14, 16, 27)
	n.poly(resurrect64.Aqua, 12, 14, 20, 14, 16, 27)
	if frame%2 == 0 {
		n.circle(11, 11, 1.2, resurrect64.White)
	} else {
		n.circle(21, 11, 1.2, resurrect64.White)
	}
}

// drawSack is a plump money bag: a flared top, a neck tied with cord, and a coin
// on its belly.
func drawSack(n pen) {
	n.circle(16, 21.5, 10.5, resurrect64.Clay)
	n.circle(15, 20.5, 9.5, resurrect64.Sand)
	n.poly(resurrect64.Sand, 12, 11, 20, 11, 22, 15, 10, 15)
	n.poly(resurrect64.Clay, 13, 8, 19, 8, 18.5, 10.5, 13.5, 10.5)
	n.poly(resurrect64.Sand, 9.5, 1.5, 12.5, 3, 16, 1, 19.5, 3, 22.5, 1.5, 19.5, 8, 12.5, 8)
	n.rect(11.5, 8.5, 9, 2.8, 1.2, resurrect64.Bark)
	n.line(19.5, 10, 23, 13.5, 1.2, resurrect64.Bark)
	n.circle(23.2, 13.8, 1.1, resurrect64.Bark)
	n.circle(11, 16, 1.6, resurrect64.White)
	n.circle(16, 21.5, 6, resurrect64.Amber)
	n.circle(16, 21.5, 4.8, resurrect64.Gold)
	for _, r := range [...][4]float64{{13.6, 18.2, 4.8, 1.3}, {13.6, 18.2, 1.3, 3.5}, {13.6, 20.9, 4.8, 1.3}, {17.1, 20.9, 1.3, 3.6}, {13.6, 23.5, 4.8, 1.3}, {15.4, 17, 1.2, 9}} {
		n.rect(r[0], r[1], r[2], r[3], 0.3, resurrect64.Rust)
	}
}

// drawGold is a heap of rough nuggets, each a lump of uneven facets.
func drawGold(n pen) {
	nugget := func(dx, dy, r float64) {
		p := func(x, y float64) (float64, float64) { return dx + x*r, dy + y*r }
		x0, y0 := p(-1, 0.2)
		x1, y1 := p(-0.55, -0.75)
		x2, y2 := p(0.25, -1)
		x3, y3 := p(1, -0.25)
		x4, y4 := p(0.7, 0.8)
		x5, y5 := p(-0.45, 0.9)
		mx, my := p(-0.1, -0.1)
		n.poly(resurrect64.Amber, x0, y0, x1, y1, x2, y2, x3, y3, x4, y4, x5, y5)
		n.poly(resurrect64.Gold, x0, y0, x1, y1, x2, y2, mx, my)
		n.poly(resurrect64.Lemon, x1, y1, x2, y2, mx, my)
		n.poly(resurrect64.Orange, mx, my, x4, y4, x5, y5)
	}
	nugget(16, 12, 5.5)
	nugget(9, 18, 6)
	nugget(23, 18.5, 6)
	nugget(5, 26, 4.5)
	nugget(15.5, 25, 6.5)
	nugget(26.5, 26, 4.5)
	n.circle(14, 8.5, 0.9, resurrect64.White)
	n.circle(21, 15, 0.8, resurrect64.White)
}

// Where the dwarf's pick-axe is: on their back, or swung up or down to dig.
const (
	pickOnBack = iota
	pickUp
	pickDown
)

// drawPick draws a pick-axe from the handle's end at (x0, y0) to its head at
// (x1, y1), whose blade runs from (ax, ay) to (bx, by).
func drawPick(n pen, x0, y0, x1, y1, ax, ay, bx, by float64) {
	n.line(x0, y0, x1, y1, 2.6, resurrect64.Caramel)
	n.line(ax, ay, bx, by, 3, resurrect64.Mist)
}

// drawDwarf is the hero: a domed helmet over a round-nosed face, and a full beard
// with a moustache, tapering to a point over a belted tunic.
// drawDwarf is the hero: a domed helmet over a round-nosed face, and a full beard
// with a moustache, tapering to a point over a belted tunic.
func drawDwarf(n pen, frame, pick int) {
	if pick == pickOnBack {
		drawPick(n, 8, 24, 4, 7, 0, 9, 9, 3.5)
	}
	stride := 1.5 * math.Copysign(1, float64(frame%2)-0.5)
	n.rect(8+stride, 26, 7.5, 5, 1.5, resurrect64.Clay)
	n.rect(17-stride, 26, 7.5, 5, 1.5, resurrect64.Clay)
	n.rect(7.5, 15, 17, 12.5, 3.5, resurrect64.Cobalt)
	n.rect(7.5, 23, 17, 2.6, 0.8, resurrect64.Bark)
	n.rect(14, 22.6, 4, 3.4, 0.8, resurrect64.Gold)
	n.rect(22.5, 15.5, 5, 7, 2.2, resurrect64.Cobalt)
	n.circle(25.5, 22, 2.6, resurrect64.Blush)
	n.circle(16, 9.5, 7.5, resurrect64.Blush)
	n.poly(resurrect64.Copper, 8.5, 9, 10.5, 12, 22, 13, 23.5, 12, 23, 18, 20, 21.5, 16.5, 25, 13, 21.5, 9.5, 17.5)
	n.line(12, 14.5, 14.5, 21, 0.9, resurrect64.Rust)
	n.line(16.5, 15, 16.5, 23, 0.9, resurrect64.Rust)
	n.line(21, 14.5, 18.5, 21, 0.9, resurrect64.Rust)
	n.circle(15.5, 12.8, 2.4, resurrect64.Caramel)
	n.circle(20.5, 12.8, 2.4, resurrect64.Caramel)
	n.circle(23, 10.8, 2.2, resurrect64.Apricot)
	n.circle(20.5, 9.9, 1.1, resurrect64.Ink)
	n.rect(6.5, 6, 19.5, 2.8, 1, resurrect64.Slate)
	n.poly(resurrect64.Fog, 8, 6.5, 9, 3, 12.5, 0.8, 19.5, 0.8, 23, 3, 24.5, 6.5)
	n.circle(12.5, 3.5, 1, resurrect64.White)
	switch pick {
	case pickUp:
		drawPick(n, 24, 20, 27, 5, 21, 8, 32, 2)
	case pickDown:
		drawPick(n, 24, 18, 29, 24, 25, 30, 32, 19)
	}
}

type tones struct{ deep, shade, skin, hi, cloth, clothS, clothH, eye int }

func koboldTones(afraid bool) tones {
	if afraid {
		return tones{resurrect64.Grape, resurrect64.Violet, resurrect64.Lavender, resurrect64.Orchid, resurrect64.Blush, resurrect64.Mauve, resurrect64.White, resurrect64.White}
	}
	return tones{resurrect64.Maroon, resurrect64.Brick, resurrect64.Vermilion, resurrect64.Coral, resurrect64.Honey, resurrect64.Caramel, resurrect64.Blush, resurrect64.Gold}
}

func goblinTones(afraid bool) tones {
	if afraid {
		return tones{resurrect64.Grape, resurrect64.Violet, resurrect64.Lavender, resurrect64.Orchid, resurrect64.Plum, resurrect64.Shadow, resurrect64.Mauve, resurrect64.White}
	}
	return tones{resurrect64.Pine, resurrect64.Forest, resurrect64.Emerald, resurrect64.Leaf, resurrect64.Grape, resurrect64.Plum, resurrect64.Violet, resurrect64.Lemon}
}

// drawKobold: a small, hunched lizard-dog with a long snout, swept-back horns
// and a tail. It stands well short of the cell, so growing is obvious.
func drawKobold(n pen, frame int, afraid bool) {
	t := koboldTones(afraid)
	step := 1.2 * math.Copysign(1, float64(frame%2)-0.5)

	// Tail, tapering behind it, with a flick.
	n.poly(t.deep, 12, 22, 12, 28.5, 6, 28, 1.5, 24.5, 5.5, 25.5)
	n.poly(t.shade, 12, 23, 12, 27, 6.5, 26.8, 3.5, 25, 6, 25.2)

	// Far leg, then near leg, with pale claws.
	n.rect(16-step, 26, 5, 5, 1.5, t.deep)
	n.rect(10+step, 25.5, 6, 5.5, 2, t.shade)
	n.rect(10+step, 28.8, 7.5, 2.2, 1, t.deep)
	for _, x := range [...]float64{15, 16.6} {
		n.poly(resurrect64.Mist, x+step, 29, x+1.5+step, 31, x+step, 31)
	}

	// Body, hunched: dark rim, scales, pale belly.
	n.rect(9, 17, 14, 11, 5.5, t.deep)
	n.rect(9.5, 17, 12.5, 9.8, 5, t.shade)
	n.rect(10, 17, 10, 7, 4, t.skin)
	n.rect(17, 19.5, 5.5, 8, 2.6, t.clothS)
	n.rect(18, 20, 4, 6.6, 2, t.cloth)
	n.line(18.2, 22, 22, 22, 0.5, t.clothS)
	n.line(18.2, 24.2, 22, 24.2, 0.5, t.clothS)

	// Spines down its back.
	for _, s := range [...][2]float64{{11.5, 16}, {9.8, 18.8}, {9, 22}} {
		n.poly(t.deep, s[0]-1.4, s[1]+1.2, s[0]-2.2, s[1]-1.2, s[0]+1.4, s[1]-0.4)
	}
	// A few scales.
	for _, s := range [...][2]float64{{13, 19.5}, {15.5, 21}, {12.5, 22.5}} {
		n.circle(s[0], s[1], 0.8, t.shade)
	}

	// Horns swept back, behind the head.
	n.poly(resurrect64.Sand, 15, 9, 5.5, 3.5, 13, 12)
	n.poly(resurrect64.Clay, 15, 10.5, 6.5, 4.2, 13, 12)
	n.poly(resurrect64.Clay, 18, 7.5, 10, 2, 16, 8.5)

	// Head with a long snout, lower jaw under it.
	n.poly(t.deep, 16, 15, 28, 15.2, 27, 18.5, 18, 19.5)
	n.circle(18, 11.5, 6.2, t.shade)
	n.poly(t.shade, 19, 6.5, 29, 10.5, 30.5, 13, 29, 15.5, 16, 16.5)
	n.circle(17.5, 11, 5.4, t.skin)
	n.poly(t.skin, 19, 6.8, 28, 10.8, 29.2, 12.6, 27.8, 14.2, 17, 14.6)
	n.circle(16.5, 8.5, 1.6, t.hi)
	n.line(21, 8.6, 27, 11, 0.9, t.hi)
	n.circle(27.6, 12, 0.7, t.deep)

	// Teeth, along the jaw.
	for _, x := range [...]float64{19, 22, 25} {
		n.poly(resurrect64.White, x, 15, x+1.8, 15, x+0.9, 17.4)
	}

	// Eye: bright with a slit, and a scowl above it.
	if afraid {
		n.circle(20, 10.8, 2.6, t.eye)
		n.circle(20.8, 10.8, 0.8, resurrect64.Ink)
	} else {
		n.circle(20, 11, 2.1, t.eye)
		n.rect(20.2, 9.3, 0.9, 3.4, 0.4, resurrect64.Ink)
		n.poly(t.deep, 17, 7.8, 23.5, 9.8, 23, 11, 17.5, 9.6)
	}

	// Little arm with claws.
	n.rect(19.5, 19, 4, 5.5, 2, t.shade)
	n.circle(23.3, 24.3, 1.9, t.skin)
	for _, a := range [...]float64{0, 1.5} {
		n.poly(resurrect64.Mist, 24+a*0.6, 25, 25.2+a*0.6, 25, 25.4+a*0.8, 27.4)
	}
}

// drawGoblin is the present goblin grown up: same green skin, yellow eyes,
// purple tunic and pointed ears, but shaded, hunched, and carrying the mattock
// that says it can dig through earth.
func drawGoblin(n pen, frame int, afraid bool) {
	t := goblinTones(afraid)
	step := 1.5 * math.Copysign(1, float64(frame%2)-0.5)

	// Mattock, behind everything but the hand.
	if !afraid {
		n.line(29, 30.5, 29, 3, 1.7, resurrect64.Bark)
		n.line(29.5, 30, 29.5, 3, 0.6, resurrect64.Clay)
		n.poly(resurrect64.Dusk, 23.5, 7, 28, 1.2, 32, 1.2, 32, 4.5, 28, 5)
		n.poly(resurrect64.Fog, 24.5, 5.6, 28.3, 1.6, 31.6, 1.6, 31.6, 2.8, 28, 3.4)
	}

	// Far ear and far arm, in shade.
	n.poly(t.deep, 22.5, 6, 27.5, 3, 25, 10)
	n.rect(3.5, 16.5, 5.5, 9, 2.6, t.shade)
	n.circle(5.5, 25.5, 2.8, t.deep)
	for _, x := range [...]float64{2.8, 5, 7.2} {
		n.poly(resurrect64.Mist, x-0.9, 26.8, x+0.9, 26.8, x, 29.4)
	}

	// Legs and boots.
	n.rect(10+step, 24, 6, 5, 1.6, t.clothS)
	n.rect(18-step, 24, 6, 5, 1.6, t.clothS)
	n.rect(8.5+step, 27.5, 8.5, 3.8, 1.5, t.deep)
	n.rect(17-step, 27.5, 8.5, 3.8, 1.5, t.deep)
	n.rect(9.5+step, 27.8, 5, 1, 0.5, t.shade)
	n.rect(18-step, 27.8, 5, 1, 0.5, t.shade)

	// Tunic: shaded, patched, ragged hem.
	n.rect(7.5, 15, 18.5, 12, 5, t.clothS)
	n.rect(7.5, 15, 16.5, 10.5, 5, t.cloth)
	n.line(10.6, 17.5, 10.6, 22, 1.4, t.clothH)
	n.poly(t.clothS, 9, 25, 12, 25, 10.5, 28.5)
	n.poly(t.clothS, 14, 25, 18, 25, 16, 29)
	n.poly(t.clothS, 20, 25, 24, 25, 22.5, 28)
	n.rect(16.5, 17.5, 4, 3.4, 0.5, t.clothS)
	n.rect(7.5, 22.8, 18.5, 2, 0.8, resurrect64.Clay)
	n.rect(7.5, 22.8, 18.5, 1, 0.5, resurrect64.Sand)
	n.rect(14, 22.3, 3, 3, 0.6, resurrect64.Fog)

	// Hunched shoulders.
	n.circle(10.5, 16.5, 3.6, t.shade)
	n.circle(23, 16.5, 3.6, t.shade)

	// Near ear, long and swept back.
	n.poly(t.deep, 11, 6.5, 0.5, 1.5, 3.2, 10.5, 11, 13.5)
	n.poly(t.skin, 11, 7.5, 2.2, 3, 4.2, 9.8, 11, 12.2)
	n.poly(t.deep, 10.5, 8.8, 5, 5.5, 6, 9.2, 10.5, 11)
	n.line(3.2, 3.8, 7, 5.8, 0.8, t.hi)

	// Head: dark edge, skin, highlight.
	n.circle(17.5, 10.2, 8.4, t.deep)
	n.circle(17.3, 9.8, 7.8, t.shade)
	n.circle(16.8, 9, 7.1, t.skin)
	n.line(12.5, 5.5, 16, 3.6, 1.5, t.hi)

	// Eyes.
	if afraid {
		n.circle(15, 8.8, 2.8, t.eye)
		n.circle(21.4, 8.8, 2.4, t.eye)
		n.circle(15.8, 9, 0.9, resurrect64.Ink)
		n.circle(22.1, 9, 0.9, resurrect64.Ink)
	} else {
		n.circle(15, 9.2, 2.4, t.eye)
		n.circle(21.4, 9.2, 2.1, t.eye)
		n.circle(15.9, 9.4, 1.1, resurrect64.Ink)
		n.circle(22.2, 9.4, 1, resurrect64.Ink)
		n.poly(t.deep, 11.8, 5.6, 18.2, 8, 24.4, 5.6, 24.4, 7, 18.2, 9.3, 11.8, 7)
	}

	// Mouth with a jutting tusk, under the hooked nose.
	if afraid {
		n.rect(12.5, 15.5, 9, 3, 1.4, resurrect64.Ink)
	} else {
		n.poly(resurrect64.Ink, 10.5, 15.6, 22.5, 15.6, 21, 18.2, 12, 18.2)
	}
	n.poly(resurrect64.White, 12.3, 18.2, 15.2, 18.2, 13.7, 14.2)

	// Hooked nose.
	n.poly(t.deep, 18, 11.8, 26.5, 12.6, 29, 16.6, 26, 17, 19.5, 15.4)
	n.poly(t.skin, 18, 11.4, 25.8, 12.2, 27.8, 15.2, 25.2, 15.3, 19.5, 14.2)
	n.line(20, 12.6, 25, 13.2, 0.8, t.hi)

	// Near arm and hand, gripping the mattock.
	n.rect(22.5, 16, 5.5, 8, 2.5, t.shade)
	n.rect(22.5, 16, 4.5, 6.5, 2, t.skin)
	n.circle(28, 23, 3, t.skin)
	n.circle(28.5, 23.8, 2.1, t.shade)
	n.rect(26.8, 21.6, 3.6, 1, 0.5, t.hi)
}

// drawMushroom is the bonus: a glowing cap on a pale stem, its glow pulsing.
func drawMushroom(n pen, frame int) {
	glow := 15.0
	if frame%2 == 0 {
		glow = 13.5
	}
	n.circle(16, 13, glow, resurrect64.Plum)
	n.rect(12, 16, 8, 12, 2.5, resurrect64.Sand)
	n.rect(12, 25, 8, 3, 1.5, resurrect64.Clay)
	n.poly(resurrect64.Violet, 4, 18, 5, 10, 10, 5, 16, 3, 22, 5, 27, 10, 28, 18)
	n.rect(4, 15, 24, 4, 2, resurrect64.Violet)
	n.circle(11, 10, 2.6, resurrect64.Lavender)
	n.circle(20, 8, 2, resurrect64.White)
	n.circle(22, 14, 2.4, resurrect64.Lavender)
}

// headstonePen is the pen for the headstone in the dwarf's cell of the given
// size, as far below its place as it still has to rise. It takes the dwarf's
// place but not the way they faced, so that RIP always reads forwards.
func (p *Play) headstonePen(dwarf pen, cell int) pen {
	stone := dwarf
	stone.mirror = false
	stone.y += float64(p.headstoneDrop(cell))
	return stone
}

// drawHeadstone is the dwarf's grave: a round-topped stone reading RIP, with grass
// around it, on a
// mound of earth.
func drawHeadstone(n pen) {
	n.circle(16, 28, 3.2, resurrect64.Bark)
	n.rect(1, 26.5, 30, 5, 2.4, resurrect64.Bark)
	n.circle(16, 12, 11.5, resurrect64.Dusk)
	n.rect(4.5, 12, 23, 16, 1, resurrect64.Dusk)
	n.circle(16, 12, 10.2, resurrect64.Fog)
	n.rect(5.8, 12, 20.4, 14.8, 0.5, resurrect64.Fog)
	for _, r := range [...][4]float64{
		{7, 10, 1.6, 9}, {7, 10, 5, 1.6}, {10.6, 10, 1.6, 4.8}, {7, 13.6, 5, 1.6}, {10.6, 15, 1.6, 4},
		{14.6, 10, 1.6, 9},
		{18.6, 10, 1.6, 9}, {18.6, 10, 5.6, 1.6}, {22.6, 10, 1.6, 5}, {18.6, 13.4, 5.6, 1.6},
	} {
		n.rect(r[0], r[1], r[2], r[3], 0.3, resurrect64.Ink)
	}
	// Blades of grass: the far ones darker, then the near ones over the stone's foot.
	for _, b := range [...][3]float64{{0.5, 21, -0.5}, {3, 23, 0.8}, {29, 23, -0.8}, {31.5, 21, 0.5}, {9, 26, -0.8}, {16, 26, 0.8}, {23, 26, -0.8}} {
		drawBlade(n, b[0], b[1], b[2], resurrect64.Forest)
	}
	for _, b := range [...][3]float64{{1.5, 24, 1}, {5.5, 26, -1}, {12, 27.5, 1}, {19.5, 27.5, -1}, {26.5, 26, 1}, {30.5, 24, -1}} {
		drawBlade(n, b[0], b[1], b[2], resurrect64.Emerald)
	}
}

// drawBlade is a blade of grass growing from the foot of the cell at x, up to
// height y, its tip leaning by lean.
func drawBlade(n pen, x, y, lean float64, c int) {
	n.poly(c, x-1.4, 32, x+1.4, 32, x+lean, y)
}

func drawShot(dst *ebiten.Image, x, y, r float64) {
	shapes.FillCircle(dst, x, y, r, resurrect64.Amber)
	shapes.FillCircle(dst, x, y, r/2, resurrect64.White)
}
