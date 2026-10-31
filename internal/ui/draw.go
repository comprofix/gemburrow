package ui

import (
	"image"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/gameskeleton/internal/gfx"
)

// LineGap is the space between menu lines, in pixels.
const LineGap = 4

// SliderCells is a slider's width, in character cells of the menu's font.
const SliderCells = 10

// Size returns the menu's width and height in font's pixels, including the
// cursor's column.
func (m *Menu) Size(font gfx.Font) (w, h int) {
	col := m.sliderColumn(font)
	for i, it := range m.Items {
		if it.Slider != nil {
			w = max(w, col+(SliderCells+1)*font.CellWidth()+font.Width("100%"))
		} else {
			w = max(w, font.Width(m.Text(i)))
		}
	}
	w += 2 * font.CellWidth() // the "> " cursor
	if m.Scrolls() {
		w += 2 * font.CellWidth() // the " ^" marks
	}
	return w, m.Shown()*(font.LineHeight()+LineGap) - LineGap
}

// Draw draws the menu with its top-left corner at (x, y): disabled items in
// sc.Dim, the selected item in sc.Highlight after a "> " cursor. Sliders
// line up in one column, each followed by its percentage. A menu that
// scrolls shows the items around the cursor, with a "^" at the right of
// the top line while there are more above, and a "v" at the bottom while
// there are more below.
func (m *Menu) Draw(dst *ebiten.Image, font gfx.Font, x, y int, sc gfx.Scheme) {
	col := m.sliderColumn(font)
	first, n := m.First(), m.Shown()
	if m.Scrolls() {
		w, _ := m.Size(font)
		mx, line := x+w-font.CellWidth(), font.LineHeight()+LineGap
		if first > 0 {
			font.DrawShadowed(dst, "^", mx, y, sc.Dim, sc.Shadow)
		}
		if first+n < len(m.Items) {
			font.DrawShadowed(dst, "v", mx, y+(n-1)*line, sc.Dim, sc.Shadow)
		}
	}
	for i := first; i < first+n; i++ {
		c := sc.Text
		switch {
		case m.Items[i].IsDisabled():
			c = sc.Dim
		case i == m.Cursor:
			c = sc.Highlight
			font.DrawShadowed(dst, ">", x, y, c, sc.Shadow)
		}
		tx := x + 2*font.CellWidth()
		if s := m.Items[i].Slider; s != nil {
			v := min(max(s(), 0), 1)
			font.DrawShadowed(dst, m.Items[i].Label, tx, y, c, sc.Shadow)
			bx := tx + col
			drawSlider(dst, bx, y, SliderCells*font.CellWidth(), font.LineHeight(), v, c, sc)
			font.DrawShadowed(dst, percent(v), bx+(SliderCells+1)*font.CellWidth(), y, c, sc.Shadow)
		} else {
			font.DrawShadowed(dst, m.Text(i), tx, y, c, sc.Shadow)
		}
		y += font.LineHeight() + LineGap
	}
}

// sliderColumn is where sliders start, after the longest slider label and a
// space, relative to the labels.
func (m *Menu) sliderColumn(font gfx.Font) int {
	col := 0
	for _, it := range m.Items {
		if it.Slider != nil {
			col = max(col, font.Width(it.Label)+font.CellWidth())
		}
	}
	return col
}

// drawSlider draws a w×h bar at (x, y) filled to v (0–1) in colour c: an
// outline with the fill one pixel inside it, and a drop shadow like text.
func drawSlider(dst *ebiten.Image, x, y, w, h int, v float64, c int, sc gfx.Scheme) {
	d := max(1, h/8) // as Font.DrawShadowed
	rect := func(x, y, w, h, colour int) {
		vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), gfx.RGBA(colour), false)
	}
	h-- // leave room for the shadow, like the glyphs
	rect(x+d, y+d, w, h, sc.Shadow)
	rect(x, y, w, h, c)
	rect(x+1, y+1, w-2, h-2, sc.Panel)
	if fill := int(math.Round(v * float64(w-4))); fill > 0 {
		rect(x+2, y+2, fill, h-4, c)
	}
}

// percent is v (0–1) as a whole percentage, e.g. "60%".
func percent(v float64) string {
	return strconv.Itoa(int(math.Round(v*100))) + "%"
}

// DrawCentred draws the menu centred across dst with its top at y.
func (m *Menu) DrawCentred(dst *ebiten.Image, font gfx.Font, y int, sc gfx.Scheme) {
	w, _ := m.Size(font)
	m.Draw(dst, font, centredX(dst.Bounds().Dx(), w), y, sc)
}

// centredX is the x that centres something w pixels wide across a screen
// screenW wide.
func centredX(screenW, w int) int { return (screenW - w) / 2 }

// Panel fills r with palette colour fill inside a one-pixel border.
func Panel(dst *ebiten.Image, r image.Rectangle, fill, border int) {
	x, y := float32(r.Min.X), float32(r.Min.Y)
	w, h := float32(r.Dx()), float32(r.Dy())
	vector.FillRect(dst, x, y, w, h, gfx.RGBA(border), false)
	vector.FillRect(dst, x+1, y+1, w-2, h-2, gfx.RGBA(fill), false)
}

// PanelAround is the rectangle of a panel with pad pixels of margin around
// a w×h block centred across a screen screenW wide with its top at y.
func PanelAround(screenW, w, h, y, pad int) image.Rectangle {
	x := centredX(screenW, w)
	return image.Rect(x-pad, y-pad, x+w+pad, y+h+pad)
}

// TextCentred draws s with a shadow, centred across dst.
func TextCentred(dst *ebiten.Image, font gfx.Font, s string, y, c, shadow int) {
	font.DrawShadowed(dst, s, font.CentreX(dst, s), y, c, shadow)
}

// Blink is true for the first half of every period ticks, for blinking
// prompts.
func Blink(tick int64, period int64) bool {
	return tick%period < period/2
}
