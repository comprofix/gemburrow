// Package world is Dot Dash's rules: a grid where the player
// steps between cells collecting dots against a clock while hazards slide
// back and forth. It has no graphics or audio: the play scene draws the
// state and plays sounds for the events Update returns.
package world

import (
	"math/rand/v2"

	"github.com/comprofix/gameskeleton/internal/session"
)

// Point is a cell on the board.
type Point struct{ X, Y int }

// Add is p moved by d.
func (p Point) Add(d Point) Point { return Point{p.X + d.X, p.Y + d.Y} }

// Dir is a direction the player is pushing.
type Dir int

const (
	None Dir = iota
	Up
	Down
	Left
	Right
)

// Delta is the step one cell in direction d.
func (d Dir) Delta() Point {
	switch d {
	case Up:
		return Point{0, -1}
	case Down:
		return Point{0, 1}
	case Left:
		return Point{-1, 0}
	case Right:
		return Point{1, 0}
	}
	return Point{}
}

// Event is something that happened during a tick, for sounds and effects.
type Event int

const (
	Stepped     Event = iota
	Picked            // a dot collected
	Bumped            // a hazard hit the player
	TimeUp            // the clock ran out: the game is over
	OutOfHearts       // the last heart was lost: the game is over
)

// Mover is something that moves cell by cell. Prev and Since let the
// scene slide it smoothly from its last cell.
type Mover struct {
	Pos, Prev Point
	Since     int // ticks since it last moved
}

func (m *Mover) moveTo(p Point) { m.Prev, m.Pos, m.Since = m.Pos, p, 0 }

// Hazard slides along a row or column, turning back at the edges.
type Hazard struct {
	Mover
	Dir  Point
	wait int // ticks until its next move
}

// World is one game of Dot Dash.
type World struct {
	Cols, Rows int
	Tuning     Tuning
	Session    *session.Session // the score and the hearts (Lives)

	Player    Mover
	FacesLeft bool
	Dots      []Point
	Hazards   []Hazard
	TicksLeft int
	SafeTicks int // while above 0 the player can't be bumped
	DotsTaken int
	Over      bool

	held   Dir
	wait   int // ticks until the next step while held
	rng    *rand.Rand
	events []Event
}

// New starts a game on a cols×rows board with the player in the middle.
// The session's Lives are set to the tuning's hearts.
func New(s *session.Session, cols, rows int, t Tuning, rng *rand.Rand) *World {
	w := &World{Cols: cols, Rows: rows, Tuning: t, Session: s, TicksLeft: t.Ticks, rng: rng}
	s.Lives = t.Hearts
	mid := Point{cols / 2, rows / 2}
	w.Player = Mover{Pos: mid, Prev: mid}
	for range t.Hazards {
		p := w.freeCell(func(p Point) bool { return dist(p, mid) >= t.SafeRadius })
		d := Point{1 - 2*w.rng.IntN(2), 0}
		if w.rng.IntN(2) == 0 {
			d = Point{d.Y, d.X}
		}
		w.Hazards = append(w.Hazards, Hazard{Mover: Mover{Pos: p, Prev: p}, Dir: d, wait: 1 + w.rng.IntN(t.HazardTicks)})
	}
	for range t.Dots {
		w.Dots = append(w.Dots, w.freeCell(w.empty))
	}
	return w
}

// Update advances the game one tick with the player pushing dir, and
// returns what happened. The slice is reused by the next call. Nothing
// changes once the game is over.
func (w *World) Update(dir Dir) []Event {
	w.events = w.events[:0]
	if w.Over {
		return w.events
	}
	w.Player.Since++
	for i := range w.Hazards {
		w.Hazards[i].Since++
	}
	w.SafeTicks = max(w.SafeTicks-1, 0)

	// A new direction steps at once; holding it steps again every
	// StepTicks.
	switch {
	case dir == None:
		w.held = None
	case dir != w.held:
		w.held, w.wait = dir, 0
		fallthrough
	default:
		if w.wait--; w.wait <= 0 {
			w.step(dir)
			w.wait = w.Tuning.StepTicks
		}
	}

	for i := range w.Hazards {
		h := &w.Hazards[i]
		if h.wait--; h.wait > 0 {
			continue
		}
		h.wait = w.Tuning.HazardTicks
		next := h.Pos.Add(h.Dir)
		if !w.inside(next) {
			h.Dir = Point{-h.Dir.X, -h.Dir.Y}
			next = h.Pos.Add(h.Dir)
		}
		h.moveTo(next)
	}
	w.checkBump()

	if !w.Over {
		if w.TicksLeft--; w.TicksLeft <= 0 {
			w.TicksLeft, w.Over = 0, true
			w.events = append(w.events, TimeUp)
		}
	}
	return w.events
}

// step moves the player one cell, if the board goes that way, and picks up
// a dot there.
func (w *World) step(dir Dir) {
	switch dir {
	case Left:
		w.FacesLeft = true
	case Right:
		w.FacesLeft = false
	}
	next := w.Player.Pos.Add(dir.Delta())
	if !w.inside(next) {
		return
	}
	w.Player.moveTo(next)
	w.events = append(w.events, Stepped)
	for i, d := range w.Dots {
		if d == next {
			w.DotsTaken++
			w.Session.AddScore(w.Tuning.DotPoints)
			w.events = append(w.events, Picked)
			w.Dots[i] = w.freeCell(w.empty)
			break
		}
	}
}

// checkBump bumps the player if a hazard is on its cell or they just
// passed through each other.
func (w *World) checkBump() {
	if w.SafeTicks > 0 {
		return
	}
	p := w.Player
	for _, h := range w.Hazards {
		swapped := h.Since == 0 && p.Since == 0 && h.Pos == p.Prev && h.Prev == p.Pos
		if h.Pos == p.Pos || swapped {
			w.SafeTicks = w.Tuning.SafeTicks
			w.events = append(w.events, Bumped)
			if w.Session.LoseLife() {
				w.Over = true
				w.events = append(w.events, OutOfHearts)
			}
			return
		}
	}
}

// HeartBonus is the end-of-game bonus for the hearts left.
func (w *World) HeartBonus() int { return w.Session.Lives * w.Tuning.HeartPoints }

func (w *World) inside(p Point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < w.Cols && p.Y < w.Rows
}

// empty reports whether p has no player, dot or hazard.
func (w *World) empty(p Point) bool {
	if p == w.Player.Pos {
		return false
	}
	for _, d := range w.Dots {
		if d == p {
			return false
		}
	}
	for _, h := range w.Hazards {
		if h.Pos == p {
			return false
		}
	}
	return true
}

// freeCell picks a random cell that ok accepts. If random tries keep
// failing it takes the first such cell in reading order, and if there is
// none, the corner.
func (w *World) freeCell(ok func(Point) bool) Point {
	for range 200 {
		p := Point{w.rng.IntN(w.Cols), w.rng.IntN(w.Rows)}
		if ok(p) {
			return p
		}
	}
	for y := range w.Rows {
		for x := range w.Cols {
			if p := (Point{x, y}); ok(p) {
				return p
			}
		}
	}
	return Point{}
}

// dist is the number of king's moves between a and b.
func dist(a, b Point) int {
	return max(abs(a.X-b.X), abs(a.Y-b.Y))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
