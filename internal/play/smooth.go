package play

import (
	"fmt"
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

	// The goblins and the shot.
	for i, g := range w.Goblins {
		n := at(g.X, g.Y)
		n.mirror = g.FacesLeft
		drawGoblin(n, int(p.ticks/goblinTicks)+i, g.Troll > 0, p.fleeing())
		if debug.Enabled {
			debug.Box(dst, n.x, n.y, float64(s*config.CellSize), float64(s*config.CellSize), sc.Highlight)
		}
	}
	if shot := w.Shot; shot != nil {
		n := at(shot.X, shot.Y)
		drawShot(dst, n.x, n.y, float64(s*config.ShotSize/2))
	}

	// The dwarf, flashing once caught.
	d := w.Dwarf
	n := at(d.X, d.Y)
	n.mirror = d.FacesLeft
	if !w.Caught || (p.ticks/flashTicks)%2 == 0 {
		drawDwarf(n, (d.X+d.Y)/stepPixels)
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
			n.rect(0, 0, artCell, artCell, tunnelRoom, resurrect64.Shadow)
			if open(x+1, y) {
				n.rect(artCell/2, 0, artCell, artCell, 0, resurrect64.Shadow)
			}
			if open(x, y+1) {
				n.rect(0, artCell/2, artCell, artCell, 0, resurrect64.Shadow)
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

func drawSack(n pen) {
	n.circle(16, 21, 10, resurrect64.Clay)
	n.circle(15, 20, 9, resurrect64.Sand)
	n.rect(13, 6, 6, 7, 1.5, resurrect64.Sand)
	n.rect(11.5, 10, 9, 2.5, 1, resurrect64.Amber)
	n.poly(resurrect64.Sand, 11, 3, 21, 3, 19, 7, 13, 7)
	n.circle(16, 21, 4, resurrect64.Gold)
	n.circle(16, 21, 2, resurrect64.Amber)
}

func drawGold(n pen) {
	for _, c := range [...][3]float64{{8, 25, 4}, {16, 25, 4.5}, {24, 25, 4}, {12, 19, 4}, {20, 19, 4}, {16, 13, 3.5}} {
		n.circle(c[0], c[1], c[2], resurrect64.Amber)
		n.circle(c[0]-0.5, c[1]-0.5, c[2]-1.2, resurrect64.Gold)
	}
	n.circle(14, 11.5, 0.9, resurrect64.White)
}

func drawDwarf(n pen, frame int) {
	stride := 1.5 * math.Copysign(1, float64(frame%2)-0.5)
	n.rect(9+stride, 26, 7, 5, 1.5, resurrect64.Ink)
	n.rect(17-stride, 26, 7, 5, 1.5, resurrect64.Ink)
	n.rect(8, 15, 16, 13, 3, resurrect64.Cobalt)
	n.rect(22, 16, 6, 4, 2, resurrect64.Blush)
	n.circle(16, 10, 7.5, resurrect64.Blush)
	n.poly(resurrect64.Orange, 9, 11, 25, 11, 23, 22, 16, 24, 9, 21)
	n.poly(resurrect64.Amber, 12, 14, 24, 14, 22, 20, 16, 21, 12, 19)
	n.circle(21, 9, 1.2, resurrect64.Ink)
	n.rect(7, 6, 18, 3, 1, resurrect64.Slate)
	n.poly(resurrect64.Fog, 8, 7, 9, 3, 13, 1, 19, 1, 23, 3, 24, 7)
}

func drawGoblin(n pen, frame int, digging, afraid bool) {
	skin, shade, cloth := resurrect64.Emerald, resurrect64.Forest, resurrect64.Grape
	if digging {
		skin, shade, cloth = resurrect64.Orange, resurrect64.Rust, resurrect64.Maroon
	}
	if afraid {
		skin, shade, cloth = resurrect64.Lavender, resurrect64.Violet, resurrect64.Grape
	}
	step := 1.5 * math.Copysign(1, float64(frame%2)-0.5)
	n.rect(9+step, 26, 6, 4, 1.5, shade)
	n.rect(17-step, 26, 6, 4, 1.5, shade)
	n.rect(10, 18, 12, 10, 3, cloth)
	n.poly(skin, 9, 12, 1, 5, 11, 7)
	n.poly(skin, 23, 12, 31, 5, 21, 7)
	n.circle(16, 12, 8, skin)
	n.circle(13, 11, 2.2, resurrect64.Lemon)
	n.circle(19, 11, 2.2, resurrect64.Lemon)
	n.circle(14, 11, 1, resurrect64.Ink)
	n.circle(20, 11, 1, resurrect64.Ink)
	n.line(12, 16, 20, 16, 1.2, resurrect64.Ink)
	if digging {
		n.line(22, 22, 29, 18, 2, shade)
		n.line(29, 18, 29, 24, 2, resurrect64.Fog)
	}
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

func drawShot(dst *ebiten.Image, x, y, r float64) {
	shapes.FillCircle(dst, x, y, r, resurrect64.Amber)
	shapes.FillCircle(dst, x, y, r/2, resurrect64.White)
}
