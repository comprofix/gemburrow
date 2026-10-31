package shapes

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/gfx"
	"github.com/comprofix/gameskeleton/internal/input"
)

// None is a colour left out: a PanelStyle's Light or Shadow set to None is
// not drawn.
const None = -1

// PanelStyle is how a panel looks: a rounded rectangle filled with Fill
// inside a Border, BorderWidth wide. Shading is a drop shadow in Shadow,
// Depth pixels down and right, and a line of Light just inside the top
// edge, as if lit from above.
type PanelStyle struct {
	Fill, Border, Light, Shadow int
	Radius, BorderWidth, Depth  float64
}

// PanelFor is a panel in the scheme's colours, sized for a unit of size
// pixels (a line of the small font suits): corners rounded by a third of
// it, a border a twelfth (at least one pixel) and a shadow a sixth.
func PanelFor(sc gfx.Scheme, size float64) PanelStyle {
	return PanelStyle{
		Fill:        sc.Panel,
		Border:      sc.Border,
		Light:       None,
		Shadow:      sc.Shadow,
		Radius:      size / 3,
		BorderWidth: max(1, math.Round(size/12)),
		Depth:       math.Round(size / 6),
	}
}

// Panel draws a panel filling r: its shadow, fill, light and border.
func Panel(dst *ebiten.Image, r Rect, s PanelStyle) {
	if s.Shadow != None && s.Depth > 0 {
		FillRoundRect(dst, r.Offset(s.Depth, s.Depth), s.Radius, s.Shadow)
	}
	if s.BorderWidth > 0 {
		FillRoundRect(dst, r, s.Radius, s.Border)
	}
	in := r.Inset(s.BorderWidth)
	FillRoundRect(dst, in, s.Radius-s.BorderWidth, s.Fill)
	if s.Light != None {
		// Along the top, just under the border and clear of the rounded
		// corners, its edges on whole pixels so it stays sharp.
		rad := in.Radius(s.Radius - s.BorderWidth)
		w := max(1, s.BorderWidth)
		y := in.Y + w/2
		Line(dst, in.X+rad, y, in.X+in.W-rad, y, w, s.Light)
	}
}

// State is how a button is shown.
type State int

const (
	Normal   State = iota
	Hover          // the pointer is over it
	Pressed        // the mouse button is held over it
	Disabled       // it can't be used now
)

// Button is a clickable rectangle with a label.
type Button struct {
	Rect
	Label    string
	Disabled bool
}

// State is how b should be shown with the pointer where in has it.
func (b Button) State(in *input.Input) State {
	if b.Disabled {
		return Disabled
	}
	p, ok := in.Pointer()
	switch {
	case !ok || !b.Contains(p):
		return Normal
	case in.MouseHeld():
		return Pressed
	default:
		return Hover
	}
}

// Clicked reports whether b was clicked this tick: the left button went
// down with the pointer over it, as menus choose on a click.
func (b Button) Clicked(in *input.Input) bool {
	p, ok := in.Pointer()
	return !b.Disabled && ok && in.Clicked() && b.Contains(p)
}

// ButtonStyle is how a button looks in each state: its panel and the
// colour of its label. A pressed button is drawn pushed in, Depth pixels
// down and right, with no shadow.
type ButtonStyle struct {
	Panel [Disabled + 1]PanelStyle
	Text  [Disabled + 1]int
}

// ButtonFor is a button in the scheme's colours, sized as PanelFor: a
// panel with the text colour; on hover its border and label lit in the
// highlight; pressed, filled with the highlight; disabled, dimmed and
// flat.
func ButtonFor(sc gfx.Scheme, size float64) ButtonStyle {
	var s ButtonStyle
	base := PanelFor(sc, size)
	for st := range s.Panel {
		s.Panel[st] = base
	}
	s.Text = [...]int{sc.Text, sc.Highlight, sc.Shadow, sc.Dim}
	s.Panel[Hover].Border = sc.Highlight
	s.Panel[Pressed].Border = sc.Highlight
	s.Panel[Pressed].Fill = sc.Highlight
	s.Panel[Disabled].Border = sc.Dim
	s.Panel[Disabled].Shadow = None
	return s
}

// DrawButton draws b as it looks in state st, its label centred in font.
func DrawButton(dst *ebiten.Image, b Button, st State, font gfx.Font, s ButtonStyle) {
	ps, r := s.Panel[st], b.Rect
	if st == Pressed {
		r = r.Offset(ps.Depth, ps.Depth)
		ps.Shadow = None
	}
	Panel(dst, r, ps)
	cx, cy := r.Centre()
	x := int(math.Round(cx)) - font.Width(b.Label)/2
	y := int(math.Round(cy)) - font.LineHeight()/2
	font.Draw(dst, b.Label, x, y, s.Text[st])
}
