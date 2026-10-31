// Package world holds Gem Burrow's rules (DESIGN §5), with no graphics or
// sound. Positions are whole pixels within the field, whose top-left corner
// is (0, 0).
package world

import (
	"errors"
	"fmt"
	"strings"

	"github.com/comprofix/gemburrow/internal/config"
	"github.com/comprofix/gemburrow/internal/session"
)

// Dir is a direction of movement.
type Dir int

// The directions.
const (
	None Dir = iota
	Up
	Down
	Left
	Right
)

func (d Dir) delta() (dx, dy int) {
	switch d {
	case Up:
		return 0, -1
	case Down:
		return 0, 1
	case Left:
		return -1, 0
	case Right:
		return 1, 0
	}
	return 0, 0
}

func (d Dir) opposite() Dir {
	switch d {
	case Up:
		return Down
	case Down:
		return Up
	case Left:
		return Right
	case Right:
		return Left
	}
	return None
}

// Event is something that happened during an Update, for the play scene to
// turn into sound and effects.
type Event int

// The events.
const (
	Dug Event = iota
	JewelTaken
	LevelCleared
)

// Level is a parsed layout.
type Level struct {
	earth, jewels  []bool
	startX, startY int // the dwarf's start, in cells
}

// Parse reads a layout (see config.Levels).
func Parse(rows []string) (*Level, error) {
	if len(rows) != config.FieldRows {
		return nil, fmt.Errorf("world: level has %d rows, want %d", len(rows), config.FieldRows)
	}
	l := &Level{
		earth:  make([]bool, config.FieldCols*config.FieldRows),
		jewels: make([]bool, config.FieldCols*config.FieldRows),
	}
	dwarves, jewels := 0, 0
	for y, row := range rows {
		if len(row) != config.FieldCols {
			return nil, fmt.Errorf("world: level row %d has %d cells, want %d", y, len(row), config.FieldCols)
		}
		for x, c := range []byte(row) {
			i := y*config.FieldCols + x
			switch c {
			case config.LevelEarth:
				l.earth[i] = true
			case config.LevelTunnel:
			case config.LevelJewel:
				l.earth[i], l.jewels[i] = true, true
				jewels++
			case config.LevelDwarf:
				l.startX, l.startY = x, y
				dwarves++
			default:
				return nil, fmt.Errorf("world: level cell (%d, %d) is %q", x, y, c)
			}
		}
	}
	if dwarves != 1 {
		return nil, fmt.Errorf("world: level has %d dwarf starts, want 1", dwarves)
	}
	if jewels == 0 {
		return nil, errors.New("world: level has no jewels")
	}
	return l, nil
}

// Dwarf is the player's digger. X and Y are its top-left corner.
type Dwarf struct {
	X, Y      int
	Moving    Dir
	Facing    Dir  // the last direction pushed
	FacesLeft bool // the last sideways direction, for mirroring the sprite
}

// World is one level in progress.
type World struct {
	Dwarf       Dwarf
	JewelsLeft  int
	JewelsTaken int

	session *session.Session
	earth   []bool
	jewels  []bool
	events  []Event
}

// New starts a level, scoring into s.
func New(s *session.Session, l *Level) *World {
	w := &World{
		Dwarf:   Dwarf{X: l.startX * config.CellSize, Y: l.startY * config.CellSize, Facing: Right},
		session: s,
		earth:   append([]bool(nil), l.earth...),
		jewels:  append([]bool(nil), l.jewels...),
	}
	for _, j := range w.jewels {
		if j {
			w.JewelsLeft++
		}
	}
	return w
}

// Earth reports whether cell (x, y) is solid earth.
func (w *World) Earth(x, y int) bool { return w.earth[y*config.FieldCols+x] }

// Jewel reports whether cell (x, y) holds a jewel.
func (w *World) Jewel(x, y int) bool { return w.jewels[y*config.FieldCols+x] }

// Cleared reports whether the level is finished.
func (w *World) Cleared() bool { return w.JewelsLeft == 0 }

// Update advances one tick with dir pushed, and returns what happened. The
// slice is valid until the next Update.
//
// The dwarf turns only when centred on a cell, so it stays on the grid, but
// can reverse at any time. A cell is dug, and its jewel taken, once the
// dwarf's centre is inside it.
func (w *World) Update(dir Dir) []Event {
	w.events = w.events[:0]
	if w.Cleared() {
		return w.events
	}
	d := &w.Dwarf
	if d.X%config.CellSize == 0 && d.Y%config.CellSize == 0 {
		d.Moving = None
		if dir != None {
			d.face(dir)
			if w.open(dir) {
				d.Moving = dir
			}
		}
	} else if dir != None && dir == d.Moving.opposite() {
		d.face(dir)
		d.Moving = dir
	}
	dx, dy := d.Moving.delta()
	d.X += dx * config.DwarfSpeed
	d.Y += dy * config.DwarfSpeed
	w.enter(d.cell())
	return w.events
}

// cell is the cell holding the dwarf's centre.
func (d *Dwarf) cell() (x, y int) {
	return (d.X + config.CellSize/2) / config.CellSize, (d.Y + config.CellSize/2) / config.CellSize
}

func (d *Dwarf) face(dir Dir) {
	d.Facing = dir
	switch dir {
	case Left:
		d.FacesLeft = true
	case Right:
		d.FacesLeft = false
	}
}

// open reports whether the field goes on in direction dir from the dwarf,
// which must be centred on a cell. Only the field's edge stops the dwarf,
// which digs through the rest.
func (w *World) open(dir Dir) bool {
	dx, dy := dir.delta()
	x, y := w.Dwarf.X/config.CellSize+dx, w.Dwarf.Y/config.CellSize+dy
	return x >= 0 && x < config.FieldCols && y >= 0 && y < config.FieldRows
}

// enter digs cell (x, y) and takes its jewel.
func (w *World) enter(x, y int) {
	i := y*config.FieldCols + x
	if w.earth[i] {
		w.earth[i] = false
		w.events = append(w.events, Dug)
	}
	if w.jewels[i] {
		w.jewels[i] = false
		w.JewelsLeft--
		w.JewelsTaken++
		w.session.AddScore(config.JewelPoints)
		w.events = append(w.events, JewelTaken)
		if w.Cleared() {
			w.events = append(w.events, LevelCleared)
		}
	}
}

// String draws the field as a layout, for tests.
func (w *World) String() string {
	var b strings.Builder
	dx, dy := w.Dwarf.cell()
	for y := range config.FieldRows {
		for x := range config.FieldCols {
			c := byte(config.LevelTunnel)
			switch {
			case x == dx && y == dy:
				c = config.LevelDwarf
			case w.Jewel(x, y):
				c = config.LevelJewel
			case w.Earth(x, y):
				c = config.LevelEarth
			}
			b.WriteByte(c)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
