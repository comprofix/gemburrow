// Package world holds Gem Burrow's rules (DESIGN §5), with no graphics or
// sound. Positions are whole pixels within the field, whose top-left corner
// is (0, 0).
package world

import (
	"errors"
	"fmt"
	"math/rand/v2"
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

// dirs lists the directions, in the fixed order that keeps runs repeatable.
var dirs = [...]Dir{Up, Down, Left, Right}

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
	ShotFired
	GoblinShot
	DwarfCaught
	ExtraLife
)

// Level is a parsed layout.
type Level struct {
	earth, jewels  []bool
	startX, startY int // the dwarf's start, in cells
	spawnX, spawnY int // where goblins enter, in cells
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
	dwarves, spawns, jewels := 0, 0, 0
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
			case config.LevelSpawn:
				l.spawnX, l.spawnY = x, y
				spawns++
			default:
				return nil, fmt.Errorf("world: level cell (%d, %d) is %q", x, y, c)
			}
		}
	}
	if dwarves != 1 {
		return nil, fmt.Errorf("world: level has %d dwarf starts, want 1", dwarves)
	}
	if spawns != 1 {
		return nil, fmt.Errorf("world: level has %d goblin spawns, want 1", spawns)
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

// Goblin is a hunter in the tunnels. X and Y are its top-left corner.
type Goblin struct {
	X, Y      int
	Moving    Dir
	FacesLeft bool
}

// Shot is the dwarf's shot in flight. X and Y are its centre.
type Shot struct {
	X, Y int
	Dir  Dir
}

// World is one level in progress.
type World struct {
	Tuning config.Tuning

	Dwarf       Dwarf
	Goblins     []Goblin
	Shot        *Shot // nil when none is in flight
	Recharge    int   // ticks until the dwarf can fire again
	JewelsLeft  int
	JewelsTaken int
	Spawned     int // goblins that have entered, less any sent back by Respawn
	Killed      int
	Caught      bool // the dwarf was caught; nothing moves until Respawn
	Over        bool // caught with no lives left

	level     *Level
	session   *session.Session
	rng       *rand.Rand
	earth     []bool
	jewels    []bool
	spawnWait int
	cleared   bool
	events    []Event
	dist      []int // scratch: tunnel distance from the dwarf
	queue     []int // scratch: the breadth-first search's queue
}

// New starts a level, scoring into s, with its randomness from rng.
func New(s *session.Session, l *Level, t config.Tuning, rng *rand.Rand) *World {
	w := &World{
		Tuning:    t,
		level:     l,
		session:   s,
		rng:       rng,
		earth:     append([]bool(nil), l.earth...),
		jewels:    append([]bool(nil), l.jewels...),
		spawnWait: t.GoblinSpawnTicks,
		dist:      make([]int, len(l.earth)),
	}
	w.placeDwarf()
	for _, j := range w.jewels {
		if j {
			w.JewelsLeft++
		}
	}
	return w
}

func (w *World) placeDwarf() {
	w.Dwarf = Dwarf{X: w.level.startX * config.CellSize, Y: w.level.startY * config.CellSize, Facing: Right}
}

// Earth reports whether cell (x, y) is solid earth.
func (w *World) Earth(x, y int) bool { return w.earth[y*config.FieldCols+x] }

// Jewel reports whether cell (x, y) holds a jewel.
func (w *World) Jewel(x, y int) bool { return w.jewels[y*config.FieldCols+x] }

// Cleared reports whether the level is finished: every jewel taken, or every
// goblin destroyed (DESIGN D5).
func (w *World) Cleared() bool { return w.cleared }

// Update advances one tick with dir pushed and fire pressed, and returns what
// happened. The slice is valid until the next Update.
func (w *World) Update(dir Dir, fire bool) []Event {
	w.events = w.events[:0]
	if w.cleared || w.Caught {
		return w.events
	}
	w.moveDwarf(dir)
	w.Recharge = max(w.Recharge-1, 0)
	if fire && w.Recharge == 0 {
		w.fire()
	}
	w.moveShot()
	if w.checkCleared() {
		return w.events
	}
	w.spawn()
	for i := range w.Goblins {
		w.moveGoblin(&w.Goblins[i])
	}
	w.shotHits()
	if w.checkCleared() {
		return w.events
	}
	w.checkCaught()
	return w.events
}

// checkCleared finishes the level if its goal has been met.
func (w *World) checkCleared() bool {
	allShot := w.Tuning.GoblinTotal > 0 && w.Killed >= w.Tuning.GoblinTotal
	if w.JewelsLeft == 0 || allShot {
		w.cleared = true
		w.events = append(w.events, LevelCleared)
	}
	return w.cleared
}

// moveDwarf moves the dwarf. It turns only when centred on a cell, so it stays
// on the grid, but can reverse at any time. A cell is dug, and its jewel
// taken, once the dwarf's centre is inside it.
func (w *World) moveDwarf(dir Dir) {
	d := &w.Dwarf
	if centred(d.X, d.Y) {
		d.Moving = None
		if dir != None {
			d.face(dir)
			if inField(d.X/config.CellSize, d.Y/config.CellSize, dir) {
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
}

func centred(x, y int) bool { return x%config.CellSize == 0 && y%config.CellSize == 0 }

// inField reports whether the field goes on in direction dir from cell (x, y).
func inField(x, y int, dir Dir) bool {
	dx, dy := dir.delta()
	x, y = x+dx, y+dy
	return x >= 0 && x < config.FieldCols && y >= 0 && y < config.FieldRows
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
		w.addScore(config.JewelPoints)
		w.events = append(w.events, JewelTaken)
	}
}

// addScore scores points, with an extra life each time the score passes a
// multiple of config.ExtraLifeEvery (DESIGN §5).
func (w *World) addScore(points int) {
	before := w.session.Score / config.ExtraLifeEvery
	w.session.AddScore(points)
	if gained := w.session.Score/config.ExtraLifeEvery - before; gained > 0 {
		w.session.Lives += gained
		w.events = append(w.events, ExtraLife)
	}
}

// fire looses a shot from the dwarf's centre in the direction faced.
func (w *World) fire() {
	d := &w.Dwarf
	w.Shot = &Shot{X: d.X + config.CellSize/2, Y: d.Y + config.CellSize/2, Dir: d.Facing}
	w.Recharge = w.Tuning.ShotRechargeTicks
	w.events = append(w.events, ShotFired)
}

// moveShot flies the shot one tick. It ends at earth or the field's edge, or
// on the first goblin it meets.
func (w *World) moveShot() {
	s := w.Shot
	if s == nil {
		return
	}
	dx, dy := s.Dir.delta()
	s.X += dx * config.ShotSpeed
	s.Y += dy * config.ShotSpeed
	if s.X < 0 || s.Y < 0 || s.X >= config.FieldCols*config.CellSize || s.Y >= config.FieldRows*config.CellSize ||
		w.Earth(s.X/config.CellSize, s.Y/config.CellSize) {
		w.Shot = nil
		return
	}
	w.shotHits()
}

// shotHits ends the shot on any goblin it now touches. It is also run after
// the goblins move, so that one can't step through the shot.
func (w *World) shotHits() {
	s := w.Shot
	if s == nil {
		return
	}
	for i := range w.Goblins {
		g := &w.Goblins[i]
		if s.X >= g.X && s.X < g.X+config.CellSize && s.Y >= g.Y && s.Y < g.Y+config.CellSize {
			w.Goblins = append(w.Goblins[:i], w.Goblins[i+1:]...)
			w.Shot = nil
			w.Killed++
			w.addScore(config.GoblinPoints)
			w.events = append(w.events, GoblinShot)
			return
		}
	}
}

// spawn brings in the next goblin when it is due, room is left on the field,
// and the dwarf is not standing by the entrance.
func (w *World) spawn() {
	t := &w.Tuning
	if w.Spawned >= t.GoblinTotal || len(w.Goblins) >= t.GoblinsAtOnce {
		return
	}
	if w.spawnWait > 0 {
		w.spawnWait--
		return
	}
	x, y := w.Dwarf.cell()
	if abs(x-w.level.spawnX)+abs(y-w.level.spawnY) <= config.SpawnClearCells {
		return
	}
	w.Goblins = append(w.Goblins, Goblin{X: w.level.spawnX * config.CellSize, Y: w.level.spawnY * config.CellSize})
	w.Spawned++
	w.spawnWait = t.GoblinSpawnTicks
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// moveGoblin moves a goblin along the tunnels, choosing its way each time it
// is centred on a cell.
func (w *World) moveGoblin(g *Goblin) {
	if centred(g.X, g.Y) {
		g.Moving = w.chooseWay(g)
		switch g.Moving {
		case Left:
			g.FacesLeft = true
		case Right:
			g.FacesLeft = false
		}
	}
	dx, dy := g.Moving.delta()
	g.X += dx * w.Tuning.GoblinSpeed
	g.Y += dy * w.Tuning.GoblinSpeed
}

// chooseWay takes a shortest way through the tunnels to the dwarf, with ties
// broken at random. With no way there, the goblin wanders, turning back only
// at a dead end.
func (w *World) chooseWay(g *Goblin) Dir {
	w.measure()
	x, y := g.X/config.CellSize, g.Y/config.CellSize
	best, ways, choice := -1, 0, None
	back, canBack := g.Moving.opposite(), false
	for _, dir := range dirs {
		if !w.walkable(x, y, dir) {
			continue
		}
		if dir == back {
			canBack = true
		}
		dx, dy := dir.delta()
		dist := w.dist[(y+dy)*config.FieldCols+x+dx]
		if dist < 0 {
			continue
		}
		switch {
		case best < 0 || dist < best:
			best, ways, choice = dist, 1, dir
		case dist == best:
			ways++
			if w.rng.IntN(ways) == 0 {
				choice = dir
			}
		}
	}
	if best >= 0 {
		return choice
	}
	ways, choice = 0, None
	for _, dir := range dirs {
		if dir != back && w.walkable(x, y, dir) {
			ways++
			if w.rng.IntN(ways) == 0 {
				choice = dir
			}
		}
	}
	if ways == 0 && canBack {
		return back
	}
	return choice
}

// walkable reports whether a goblin can step from cell (x, y) in direction
// dir: goblins keep to the tunnels.
func (w *World) walkable(x, y int, dir Dir) bool {
	if !inField(x, y, dir) {
		return false
	}
	dx, dy := dir.delta()
	return !w.Earth(x+dx, y+dy)
}

// measure fills w.dist with each cell's distance from the dwarf through the
// tunnels, or -1 if there is no way.
func (w *World) measure() {
	for i := range w.dist {
		w.dist[i] = -1
	}
	x, y := w.Dwarf.cell()
	start := y*config.FieldCols + x
	w.dist[start] = 0
	w.queue = append(w.queue[:0], start)
	for head := 0; head < len(w.queue); head++ {
		i := w.queue[head]
		cx, cy := i%config.FieldCols, i/config.FieldCols
		for _, dir := range dirs {
			if !w.walkable(cx, cy, dir) {
				continue
			}
			dx, dy := dir.delta()
			n := (cy+dy)*config.FieldCols + cx + dx
			if w.dist[n] < 0 {
				w.dist[n] = w.dist[i] + 1
				w.queue = append(w.queue, n)
			}
		}
	}
}

// checkCaught costs the dwarf a life if a goblin is touching it. Boxes are
// inset so that only a real overlap counts.
func (w *World) checkCaught() {
	reach := config.CellSize - 2*config.ContactInset
	for i := range w.Goblins {
		g := &w.Goblins[i]
		if abs(g.X-w.Dwarf.X) < reach && abs(g.Y-w.Dwarf.Y) < reach {
			w.Caught = true
			w.Over = w.session.LoseLife()
			w.events = append(w.events, DwarfCaught)
			return
		}
	}
}

// Respawn puts the dwarf back at its start after being caught. What it dug
// and took stays as it is. The goblins on the field go back to wait their turn.
func (w *World) Respawn() {
	w.Spawned -= len(w.Goblins)
	w.Goblins = w.Goblins[:0]
	w.Shot = nil
	w.Recharge = 0
	w.Caught = false
	w.spawnWait = w.Tuning.GoblinSpawnTicks
	w.placeDwarf()
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
