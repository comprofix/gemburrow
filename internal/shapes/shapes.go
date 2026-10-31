// Package shapes draws smooth, anti-aliased shapes in palette colours, for
// art drawn in code rather than from sprites: rectangles, rounded
// rectangles, circles, lines and polygons, and from them panels and
// buttons. Every colour is a palette index, as everywhere else.
package shapes

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/gameskeleton/internal/gfx"
)

// Rect is a rectangle in screen pixels, with its top-left corner at (X, Y).
type Rect struct{ X, Y, W, H float64 }

// R is the Rect at (x, y) of size w×h.
func R(x, y, w, h float64) Rect { return Rect{x, y, w, h} }

// Contains reports whether p lies inside r: the left and top edges are in,
// the right and bottom out, so rectangles side by side never share a pixel.
func (r Rect) Contains(p image.Point) bool {
	x, y := float64(p.X), float64(p.Y)
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Inset is r shrunk by d on every side (grown, if d is negative). It never
// shrinks below nothing: past that it is an empty rectangle at the middle.
func (r Rect) Inset(d float64) Rect {
	w, h := max(r.W-2*d, 0), max(r.H-2*d, 0)
	return Rect{r.X + (r.W-w)/2, r.Y + (r.H-h)/2, w, h}
}

// Offset is r moved by (dx, dy).
func (r Rect) Offset(dx, dy float64) Rect { return Rect{r.X + dx, r.Y + dy, r.W, r.H} }

// Centre is the middle of r.
func (r Rect) Centre() (x, y float64) { return r.X + r.W/2, r.Y + r.H/2 }

// Centred is a w×h rectangle with the same middle as r.
func (r Rect) Centred(w, h float64) Rect {
	cx, cy := r.Centre()
	return Rect{cx - w/2, cy - h/2, w, h}
}

// Radius is radius limited to what fits r: no more than half its shorter
// side, and not below nothing.
func (r Rect) Radius(radius float64) float64 {
	return min(max(radius, 0), min(r.W, r.H)/2)
}

// FillRect fills r with palette colour c.
func FillRect(dst *ebiten.Image, r Rect, c int) {
	vector.FillRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), gfx.RGBA(c), true)
}

// StrokeRect outlines r in palette colour c with a line width pixels wide,
// lying inside r so the outline never spills past it.
func StrokeRect(dst *ebiten.Image, r Rect, width float64, c int) {
	in := r.Inset(width / 2)
	vector.StrokeRect(dst, float32(in.X), float32(in.Y), float32(in.W), float32(in.H), float32(width), gfx.RGBA(c), true)
}

// RoundRect is the outline of r with its corners rounded off by radius (no
// more than fits), as a path for filling or stroking.
func RoundRect(r Rect, radius float64) *vector.Path {
	x0, y0, x1, y1 := float32(r.X), float32(r.Y), float32(r.X+r.W), float32(r.Y+r.H)
	rad := float32(r.Radius(radius))
	var p vector.Path
	p.MoveTo(x0+rad, y0)
	p.LineTo(x1-rad, y0)
	p.ArcTo(x1, y0, x1, y0+rad, rad)
	p.LineTo(x1, y1-rad)
	p.ArcTo(x1, y1, x1-rad, y1, rad)
	p.LineTo(x0+rad, y1)
	p.ArcTo(x0, y1, x0, y1-rad, rad)
	p.LineTo(x0, y0+rad)
	p.ArcTo(x0, y0, x0+rad, y0, rad)
	p.Close()
	return &p
}

// FillRoundRect fills r, its corners rounded by radius, with palette
// colour c.
func FillRoundRect(dst *ebiten.Image, r Rect, radius float64, c int) {
	FillPath(dst, RoundRect(r, radius), c)
}

// StrokeRoundRect outlines r, its corners rounded by radius, in palette
// colour c with a line width pixels wide, lying inside r.
func StrokeRoundRect(dst *ebiten.Image, r Rect, radius, width float64, c int) {
	StrokePath(dst, RoundRect(r.Inset(width/2), radius-width/2), width, c)
}

// FillCircle fills the circle of radius r round (cx, cy) with palette
// colour c.
func FillCircle(dst *ebiten.Image, cx, cy, r float64, c int) {
	vector.FillCircle(dst, float32(cx), float32(cy), float32(r), gfx.RGBA(c), true)
}

// StrokeCircle outlines the circle of radius r round (cx, cy) in palette
// colour c with a line width pixels wide, lying inside the circle.
func StrokeCircle(dst *ebiten.Image, cx, cy, r, width float64, c int) {
	vector.StrokeCircle(dst, float32(cx), float32(cy), float32(r-width/2), float32(width), gfx.RGBA(c), true)
}

// Line draws a line from (x0, y0) to (x1, y1) in palette colour c, width
// pixels wide.
func Line(dst *ebiten.Image, x0, y0, x1, y1, width float64, c int) {
	vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), float32(width), gfx.RGBA(c), true)
}

// FillPoly fills the polygon whose corners are given as x, y pairs, in
// palette colour c.
func FillPoly(dst *ebiten.Image, c int, pts ...float64) {
	if len(pts) < 6 {
		return
	}
	var p vector.Path
	p.MoveTo(float32(pts[0]), float32(pts[1]))
	for i := 2; i+1 < len(pts); i += 2 {
		p.LineTo(float32(pts[i]), float32(pts[i+1]))
	}
	p.Close()
	FillPath(dst, &p, c)
}

// FillPath fills p with palette colour c.
func FillPath(dst *ebiten.Image, p *vector.Path, c int) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(gfx.RGBA(c))
	vector.FillPath(dst, p, nil, op)
}

// StrokePath draws along p in palette colour c, width pixels wide.
func StrokePath(dst *ebiten.Image, p *vector.Path, width float64, c int) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(gfx.RGBA(c))
	vector.StrokePath(dst, p, &vector.StrokeOptions{Width: float32(width), LineJoin: vector.LineJoinRound}, op)
}

// Arc is the points along a circle of radius r round (cx, cy), from angle
// a0 to a1 in radians (0 is to the right, and angles turn clockwise, as y
// grows downwards), n+1 of them, as x, y pairs for FillPoly.
func Arc(cx, cy, r, a0, a1 float64, n int) []float64 {
	n = max(n, 1)
	pts := make([]float64, 0, 2*(n+1))
	for i := range n + 1 {
		a := a0 + (a1-a0)*float64(i)/float64(n)
		pts = append(pts, cx+r*math.Cos(a), cy+r*math.Sin(a))
	}
	return pts
}
