package play

import (
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gemburrow/internal/config"
	"github.com/comprofix/gemburrow/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/gemburrow/internal/shapes"
	"github.com/comprofix/gemburrow/internal/world"
)

// The effects are worked out from the ticks alone and touch nothing in the
// world, so they use no random numbers and cannot change a game (DESIGN §7).
// Both looks draw the same motes: round in Smooth, in chunky squares in pixel art.

// How the effects are laid out, in pixels of a cell, for a kobold facing right.
const (
	puffMotes     = 10
	puffSpread    = 15.0
	puffRise      = 5.0
	puffShade     = 0.4
	puffShadeDrop = 1.5
	noseX         = 29.0
	noseY         = 12.5
	mouthX        = 29.0
	mouthY        = 16.5
	wispGap       = 12 // ticks between wisps of smoke
	wispLife      = 45
	wispDrift     = 5.0
	wispRise      = 15.0
	flameMotes    = 7
	flameReach    = 15.0
	flameGrow     = 8 // ticks for the flame to reach out, and to die back
	flameSpeed    = 0.045
	flameFlicker  = 0.6
	pixelMoteSize = 2 // pixel art's motes snap to the sprites' doubled pixels
)

// mote is a speck of smoke or flame, its centre in the coordinates it was asked for.
type mote struct {
	x, y, size float64
	c          int
}

// smokeColour fades as the smoke gets older, f running from 0 to 1.
func smokeColour(f float64) int {
	switch {
	case f < 0.35:
		return resurrect64.Mist
	case f < 0.7:
		return resurrect64.Fog
	}
	return resurrect64.Dusk
}

// puff is the burst of smoke as a kobold becomes a goblin or back, for the cell
// at (x, y), age ticks after it began.
func puff(x, y, age int) []mote {
	if age < 0 || age >= config.PuffTicks {
		return nil
	}
	f := float64(age) / config.PuffTicks
	cx, cy := float64(x)+config.CellSize/2, float64(y)+config.CellSize/2
	out := make([]mote, 0, 2*puffMotes+1)
	lobe := func(k int, spread, drop, scale float64, c int) {
		a := 2 * math.Pi * float64(k) / puffMotes
		d := spread * math.Sqrt(f+0.1)
		size := (10 + 3*float64(k%3)) * (1 - f*f) * scale
		out = append(out, mote{cx + d*math.Cos(a), cy + d*math.Sin(a) - puffRise*f + drop, size, c})
	}
	// A shaded cloud: darker lobes underneath, paler ones over them.
	for k := range puffMotes {
		lobe(k, puffSpread, puffShadeDrop, 1, smokeColour(f+puffShade))
	}
	if f < 0.5 {
		out = append(out, mote{cx, cy, 20 * (1 - f), smokeColour(f)})
	}
	for k := 0; k < puffMotes; k += 2 {
		lobe(k, puffSpread*0.7, -puffShadeDrop, 0.75, smokeColour(f))
	}
	return out
}

// smoke is the wisps rising from the nose over a window of ticks, at ph into it.
func smoke(ph, window int) []mote {
	var out []mote
	for j := 0; j*wispGap <= window-wispLife; j++ {
		age := ph - j*wispGap
		if age < 0 || age >= wispLife {
			continue
		}
		f := float64(age) / wispLife
		sway := 1.5 * math.Sin(6*f+float64(j))
		out = append(out, mote{noseX + wispDrift*f + sway, noseY - wispRise*f, 3 + 4*f, smokeColour(f)})
	}
	return out
}

// flame is the fire from the mouth, age ticks into a burst, at a game time t.
func flame(age int, t int64) []mote {
	reach := flameReach * min(float64(age)/flameGrow, float64(config.FlameTicks-age)/flameGrow, 1)
	out := make([]mote, 0, flameMotes)
	for k := range flameMotes {
		p := math.Mod(float64(k)/flameMotes+float64(t)*flameSpeed, 1)
		c := resurrect64.Vermilion
		switch {
		case p < 0.25:
			c = resurrect64.Lemon
		case p < 0.5:
			c = resurrect64.Gold
		case p < 0.75:
			c = resurrect64.Orange
		}
		y := mouthY + (1+2.5*p)*math.Sin(float64(t)*flameFlicker+float64(k)*2.1)
		out = append(out, mote{mouthX + p*reach, y, 2.5 + 3*math.Sin(p*math.Pi), c})
	}
	return out
}

// breath is the smoke and the occasional fire of a kobold at (x, y): smoke every
// cycle, and every few cycles a burst of flame if canFlame, then more smoke. The
// kobolds are staggered by their index. It harms no one (DESIGN §7).
func breath(x, y int, left bool, ticks int64, index int, canFlame bool) []mote {
	t := ticks + int64(index)*config.BreathStaggerTicks
	ph := int(t % config.BreathCycleTicks)
	fiery := canFlame && t/config.BreathCycleTicks%config.FlameEveryCycles == 0
	var out []mote
	switch nose := config.NoseSmokeTicks; {
	case ph < nose:
		out = smoke(ph, nose)
	case !fiery:
	case ph < nose+config.FlameTicks:
		out = flame(ph-nose, t)
	case ph < 2*nose+config.FlameTicks:
		out = smoke(ph-nose-config.FlameTicks, nose)
	}
	for i := range out {
		if left {
			out[i].x = config.CellSize - out[i].x
		}
		out[i].x += float64(x)
		out[i].y += float64(y)
	}
	return out
}

// firePuff is a kobold changing form: where, and when it began.
type firePuff struct {
	x, y int
	born int64
}

// updatePuffs starts a puff for each change this tick and drops the finished.
func (p *Play) updatePuffs() {
	p.puffs = slices.DeleteFunc(p.puffs, func(f firePuff) bool {
		return p.ticks-f.born >= config.PuffTicks
	})
	for _, g := range p.world.Changed {
		p.puffs = append(p.puffs, firePuff{g.X, g.Y, p.ticks})
	}
}

// flameFree reports whether a kobold has room to breathe fire: walking sideways
// with open tunnel ahead, so the flame never licks across solid earth.
func (p *Play) flameFree(g world.Kobold) bool {
	if g.Moving != world.Left && g.Moving != world.Right {
		return false
	}
	x := g.X + config.CellSize
	if g.FacesLeft {
		x = g.X - 1
	}
	if x < 0 || x >= config.FieldCols*config.CellSize {
		return false
	}
	return !p.world.Earth(x/config.CellSize, (g.Y+config.CellSize/2)/config.CellSize)
}

// motes is everything to draw over the kobolds, in field coordinates.
func (p *Play) motes() []mote {
	var out []mote
	if !p.fleeing() {
		for i, g := range p.world.Kobolds {
			if g.Goblin == 0 && !p.buried(g) {
				out = append(out, breath(g.X, g.Y, g.FacesLeft, p.ticks, i, p.flameFree(g))...)
			}
		}
	}
	for _, f := range p.puffs {
		out = append(out, puff(f.x, f.y, int(p.ticks-f.born))...)
	}
	return out
}

// drawMotes draws motes in the smooth look, with n's corner as their origin.
func drawMotes(n pen, motes []mote) {
	for _, m := range motes {
		n.circle(m.x, m.y, m.size/2, m.c)
	}
}

// drawPixelMotes draws motes in the pixel look, offset by (ox, oy).
func drawPixelMotes(dst *ebiten.Image, motes []mote, ox, oy float64) {
	snap := func(v float64) float64 { return math.Floor(v/pixelMoteSize) * pixelMoteSize }
	for _, m := range motes {
		size := max(snap(m.size), pixelMoteSize)
		shapes.FillRect(dst, shapes.R(ox+snap(m.x-size/2), oy+snap(m.y-size/2), size, size), m.c)
	}
}
