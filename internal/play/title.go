package play

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gemburrow/internal/config"
	"github.com/comprofix/gemburrow/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/gemburrow/internal/scene"
	"github.com/comprofix/gemburrow/internal/shapes"
)

// The title's logo letters, on a grid of config.TitleLetterCols by
// config.TitleLetterRows blocks.
var titleLetters = map[rune][config.TitleLetterRows]string{
	'G': {".###.", "#...#", "#....", "#.###", "#...#", "#...#", ".###."},
	'E': {"#####", "#....", "#....", "####.", "#....", "#....", "#####"},
	'M': {"#...#", "##.##", "#.#.#", "#.#.#", "#...#", "#...#", "#...#"},
	'B': {"####.", "#...#", "#...#", "####.", "#...#", "#...#", "####."},
	'U': {"#...#", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'R': {"####.", "#...#", "#...#", "####.", "#.#..", "#..#.", "#...#"},
	'O': {".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'W': {"#...#", "#...#", "#...#", "#.#.#", "#.#.#", "##.##", "#...#"},
}

// What lies in the title's cavern, as {column, row}, around the tunnel on
// config.TitleTunnelRow.
var (
	titleJewels = [][2]int{{1, 4}, {2, 0}, {7, 1}, {9, 4}, {11, 0}, {13, 3}, {16, 1}, {18, 4}}
	titleSacks  = [][2]int{{4, 1}, {14, 1}, {5, 3}}
	titleGold   = [][2]int{{16, 3}}
)

type titleKind int

const (
	titleDwarf titleKind = iota
	titleGoblin
	titleTroll
)

// titleActor is one of the cavern's characters at a moment of the loop.
type titleActor struct {
	kind   titleKind
	x      int
	left   bool // facing and running left
	scared bool
	dig    bool
}

// titleScene is the cavern at a moment, worked out from the ticks alone so that
// drawing changes nothing: how many of the tunnel's cells are dug, and who is
// in it. The dwarf first digs across with a goblin and a troll behind them, then
// chases them back, and so on through the finished tunnel.
func titleScene(ticks int64) (dug int, actors []titleActor) {
	pass := ticks / config.TitlePassTicks
	x := int(ticks%config.TitlePassTicks)*config.TitleWalkPixels - config.CellSize
	screen := config.Modern.ScreenWidth
	cols := screen / config.CellSize

	if pass%2 == 1 {
		// Back again, the other way, with the dwarf last and the others afraid.
		at := func(behind int) int { return screen - x + behind }
		return cols, []titleActor{
			{kind: titleGoblin, x: at(0), left: true, scared: true},
			{kind: titleTroll, x: at(config.TitleGapPixels), left: true, scared: true},
			{kind: titleDwarf, x: at(2 * config.TitleGapPixels), left: true},
		}
	}
	dug = cols
	digging := pass == 0
	if digging {
		dug = 0
		if centre := x + config.CellSize/2; centre >= 0 {
			dug = min(centre/config.CellSize+1, cols)
		}
	}
	return dug, []titleActor{
		{kind: titleTroll, x: x - 2*config.TitleGapPixels},
		{kind: titleGoblin, x: x - config.TitleGapPixels},
		{kind: titleDwarf, x: x, dig: digging && x < screen},
	}
}

// DrawTitle draws the title's art for either look: the logo, and the cavern with
// the dwarf digging through it (DESIGN §7). The shared title scene adds the prompt.
func DrawTitle(ctx *scene.Context, dst *ebiten.Image, ticks int64) {
	s := ctx.Preset.ScreenWidth / config.Modern.ScreenWidth
	smooth := ctx.Preset.Art == config.ArtSmooth
	drawLogo(dst, s)
	if smooth {
		drawTitleSmooth(dst, s, ticks)
		return
	}
	drawTitlePixel(ctx, dst, ticks)
}

// drawLogo draws the game's name in solid gold letters built of blocks, with a
// lit top edge and a dark foot wherever a stroke ends, over a shadow.
func drawLogo(dst *ebiten.Image, s int) {
	const name = "GEM BURROW"
	b := config.TitleBlockPixels
	cols := len(name)*(config.TitleLetterCols+1) - 1
	left := (config.Modern.ScreenWidth - cols*b) / 2
	block := func(x, y, w, h, c int) {
		shapes.FillRect(dst, shapes.R(float64(s*x), float64(s*y), float64(s*w), float64(s*h)), c)
	}
	for pass := range 2 {
		for i, r := range name {
			for row, line := range titleLetters[r] {
				for col, on := range line {
					if on != '#' {
						continue
					}
					x := left + (i*(config.TitleLetterCols+1)+col)*b
					y := config.TitleLogoY + row*b
					if pass == 0 {
						block(x+config.TitleShadowPixels, y+config.TitleShadowPixels, b, b, resurrect64.Bark)
						continue
					}
					block(x, y, b, b, resurrect64.Gold)
					if row == 0 || titleLetters[r][row-1][col] != '#' {
						block(x, y, b, config.TitleShadowPixels, resurrect64.Honey)
					}
					if row == config.TitleLetterRows-1 || titleLetters[r][row+1][col] != '#' {
						block(x, y+b-config.TitleShadowPixels, b, config.TitleShadowPixels, resurrect64.Amber)
					}
				}
			}
		}
	}
}

func drawTitleSmooth(dst *ebiten.Image, s int, ticks int64) {
	u := float64(s * config.CellSize / artCell)
	cell := s * config.CellSize
	top := s * config.TitleSceneY
	cols := config.Modern.ScreenWidth / config.CellSize
	at := func(x, y int) pen { return pen{dst: dst, x: float64(s * x), y: float64(top + s*y), u: u} }
	dug, actors := titleScene(ticks)

	shapes.FillRect(dst, shapes.R(0, float64(top), float64(cols*cell), float64(config.TitleRows*cell)), resurrect64.Rust)
	for y := range config.TitleRows {
		for x := range cols {
			drawSpecks(at(x*config.CellSize, y*config.CellSize), x, y)
		}
	}
	if dug > 0 {
		n := at(0, config.TitleTunnelRow*config.CellSize)
		shapes.FillRoundRect(dst, shapes.R(n.x-float64(cell), n.y, float64((dug+1)*cell), float64(cell)), tunnelRoom*u, resurrect64.Ink)
	}
	frame := int(ticks / jewelTicks)
	for _, j := range titleJewels {
		drawJewel(at(j[0]*config.CellSize, j[1]*config.CellSize), frame+j[0])
	}
	for _, c := range titleSacks {
		drawSack(at(c[0]*config.CellSize, c[1]*config.CellSize))
	}
	for _, c := range titleGold {
		drawGold(at(c[0]*config.CellSize, c[1]*config.CellSize))
	}
	for i, a := range actors {
		n := at(a.x, config.TitleTunnelRow*config.CellSize)
		n.mirror = a.left
		step := int(ticks/goblinTicks) + i
		switch a.kind {
		case titleTroll:
			drawTroll(n, step, a.scared)
		case titleGoblin:
			drawGoblin(n, step, a.scared)
		default:
			pick := pickOnBack
			if a.dig {
				pick = pickUp + int(ticks/digTicks)%2
			}
			drawDwarf(n, a.x/stepPixels, pick)
		}
	}
}

func drawTitlePixel(ctx *scene.Context, dst *ebiten.Image, ticks int64) {
	sheet := ctx.Sheet("gemburrow")
	earth, tunnel, jewel := sheet.Sprite("earth"), sheet.Sprite("tunnel"), sheet.Sprite("jewel")
	cols := config.Modern.ScreenWidth / config.CellSize
	at := func(c [2]int) (float64, float64) {
		return float64(c[0] * config.CellSize), float64(config.TitleSceneY + c[1]*config.CellSize)
	}
	dug, actors := titleScene(ticks)
	for y := range config.TitleRows {
		for x := range cols {
			sx, sy := at([2]int{x, y})
			if y == config.TitleTunnelRow && x < dug {
				tunnel.Draw(dst, 0, sx, sy, false)
			} else {
				earth.Draw(dst, x+y, sx, sy, false)
			}
		}
	}
	frame := int(ticks / jewelTicks)
	for _, c := range titleJewels {
		sx, sy := at(c)
		jewel.Draw(dst, frame+c[0], sx, sy, false)
	}
	for _, c := range titleSacks {
		sx, sy := at(c)
		sheet.Sprite("sack").Draw(dst, 0, sx, sy, false)
	}
	for _, c := range titleGold {
		sx, sy := at(c)
		sheet.Sprite("gold").Draw(dst, 0, sx, sy, false)
	}
	y := float64(config.TitleSceneY + config.TitleTunnelRow*config.CellSize)
	for i, a := range actors {
		name, frame := "dwarf", a.x/stepPixels
		switch {
		case a.kind == titleDwarf && a.dig:
			name, frame = "dwarf_dig", int(ticks/digTicks)
		case a.kind == titleGoblin:
			name, frame = "goblin", int(ticks/goblinTicks)+i
		case a.kind == titleTroll:
			name, frame = "troll", int(ticks/goblinTicks)+i
		}
		if a.scared {
			name += "_scared"
		}
		sheet.Sprite(name).Draw(dst, max(frame, 0), float64(a.x), y, a.left)
	}
}
