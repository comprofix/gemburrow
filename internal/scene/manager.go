package scene

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/gfx"
)

// FadeTicks is the length of each half of a transition (fade out, then fade
// in): a quarter of a second each way.
const FadeTicks = config.TPS / 4

type phase int

const (
	idle phase = iota
	fadingOut
	fadingIn
)

// Manager runs the current scene, fades between scenes and applies the
// screen shake. Scenes are not updated while a fade is in progress.
type Manager struct {
	cur, pending Scene
	phase        phase
	tick         int
	buf          *ebiten.Image // the scene drawn before shaking
}

// NewManager starts with s, fading in from the shadow colour.
func NewManager(s Scene) *Manager {
	return &Manager{cur: s, phase: fadingIn}
}

// Current returns the active scene.
func (m *Manager) Current() Scene { return m.cur }

// Update advances the current scene or the transition, and the shake.
func (m *Manager) Update(ctx *Context) error {
	ctx.Shake.update(ctx.Settings != nil && ctx.Settings.ReducedMotion)
	switch m.phase {
	case idle:
		if err := updateScene(ctx, m.cur); err != nil {
			return err
		}
		if ctx.next != nil {
			if ctx.cut {
				m.cur = ctx.next
			} else {
				m.pending = ctx.next
				m.phase, m.tick = fadingOut, 0
			}
			ctx.next, ctx.cut = nil, false
		}
	case fadingOut:
		m.tick++
		if m.tick >= FadeTicks {
			m.cur, m.pending = m.pending, nil
			m.phase, m.tick = fadingIn, 0
		}
	case fadingIn:
		m.tick++
		if m.tick >= FadeTicks {
			m.phase, m.tick = idle, 0
		}
	}
	return nil
}

// Darkness is how far the screen is faded, from 0 to 1.
func (m *Manager) Darkness() float32 {
	switch m.phase {
	case fadingOut:
		return float32(m.tick) / FadeTicks
	case fadingIn:
		return 1 - float32(m.tick)/FadeTicks
	}
	return 0
}

// shakeOffset is this tick's shake in the shown screen's pixels. A shake is
// in the pixel look's, so it moves the screen as far in either look.
func (c *Context) shakeOffset() (dx, dy int) {
	dx, dy = c.Shake.Offset()
	if px := c.Looks.Pixel(); px != nil {
		k := c.Preset.ScreenWidth / px.ScreenWidth
		dx, dy = k*dx, k*dy
	}
	return dx, dy
}

// Draw renders the current scene, shaken if a shake is running, then the
// fade.
func (m *Manager) Draw(ctx *Context, dst *ebiten.Image) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	if dx, dy := ctx.shakeOffset(); dx != 0 || dy != 0 {
		if m.buf != nil && (m.buf.Bounds().Dx() != w || m.buf.Bounds().Dy() != h) {
			m.buf.Deallocate()
			m.buf = nil
		}
		if m.buf == nil {
			m.buf = ebiten.NewImage(w, h)
		}
		m.buf.Clear()
		drawScene(ctx, m.cur, m.buf)
		dst.Fill(gfx.RGBA(ctx.Scheme.Shadow))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(dx), float64(dy))
		dst.DrawImage(m.buf, op)
	} else {
		drawScene(ctx, m.cur, dst)
	}
	if d := m.Darkness(); d > 0 {
		Dim(dst, ctx.Scheme.Shadow, d)
	}
}

// Dim covers dst with palette colour c at the given opacity (0–1).
func Dim(dst *ebiten.Image, c int, opacity float32) {
	col := gfx.RGBA(c)
	col.R = uint8(float32(col.R) * opacity)
	col.G = uint8(float32(col.G) * opacity)
	col.B = uint8(float32(col.B) * opacity)
	col.A = uint8(255 * opacity)
	vector.FillRect(dst, 0, 0, float32(dst.Bounds().Dx()), float32(dst.Bounds().Dy()), col, false)
}

// Shaker moves the whole screen back and forth for a moment, e.g. when the
// player is hit. It does nothing when reduced motion is on.
type Shaker struct {
	left, px int
}

// Start shakes the screen for ticks ticks by up to px pixels.
func (s *Shaker) Start(ticks, px int) {
	s.left, s.px = max(s.left, ticks), max(px, 1)
}

func (s *Shaker) update(reducedMotion bool) {
	if reducedMotion {
		s.left = 0
		return
	}
	s.left = max(s.left-1, 0)
}

// Offset is this tick's shift: alternating sides, and down to one pixel for
// the last few ticks so it eases off.
func (s *Shaker) Offset() (dx, dy int) {
	if s.left == 0 {
		return 0, 0
	}
	d := s.px
	if s.left <= 4 {
		d = 1
	}
	if s.left%2 == 1 {
		d = -d
	}
	return d, 0
}
