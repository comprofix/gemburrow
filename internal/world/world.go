// Package world holds Gem Burrow's rules (DESIGN §5), with no graphics or
// sound. Positions are whole pixels within the field, whose top-left corner
// is (0, 0).
package world

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
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
	JewelRun
	LevelCleared
	ShotFired
	KoboldShot
	DwarfCaught
	ExtraLife
	SackWobbling
	SackPushed
	SackLanded
	SackBurst
	KoboldCrushed
	GoldTaken
	MushroomAppeared
	MushroomTaken
	KoboldEaten
	ChaseOver
	TankardAppeared
)

// Level is a parsed layout.
type Level struct {
	earth, jewels  []bool
	startX, startY int // the dwarf's start, in cells
	spawnX, spawnY int // where kobolds enter, in cells
	sacks          [][2]int
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
			case config.LevelSack:
				l.earth[i] = true
				l.sacks = append(l.sacks, [2]int{x, y})
			default:
				return nil, fmt.Errorf("world: level cell (%d, %d) is %q", x, y, c)
			}
		}
	}
	if dwarves != 1 {
		return nil, fmt.Errorf("world: level has %d dwarf starts, want 1", dwarves)
	}
	if spawns != 1 {
		return nil, fmt.Errorf("world: level has %d kobold spawns, want 1", spawns)
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
	Pushing   bool // pushing a sack, at its speed, until next centred
	Digging   bool // cutting into earth, until next centred
}

// Kobold is a hunter in the tunnels. X and Y are its top-left corner.
type Kobold struct {
	X, Y      int
	Moving    Dir
	FacesLeft bool
	Goblin    int  // ticks left as a goblin, which can cut through earth
	Pushing   bool // pushing a sack, at its speed, until next centred

	impatience int  // ticks spent crossing other kobolds or stuck on a sack
	stuck      bool // a sack stopped it this tick
}

// SackState is what a sack is doing.
type SackState int

// The sack states.
const (
	SackResting SackState = iota
	SackWobble
	SackSliding
	SackFalling
)

// Sack is a treasure sack (DESIGN §6, D6). X and Y are its top-left corner.
type Sack struct {
	X, Y   int
	State  SackState
	Wobble int  // ticks left of wobbling
	Slide  Dir  // the way it is being pushed
	Fallen int  // pixels fallen so far
	gone   bool // to be removed at the end of moveSacks
}

// Gold is a burst sack's treasure, lying in a cell for a limited time.
type Gold struct {
	X, Y  int // its cell
	Ticks int // ticks left
}

// Shot is the dwarf's shot in flight. X and Y are its centre.
type Shot struct {
	X, Y int
	Dir  Dir
}

// Pickup is an item that appears once in a level, when enough jewels are gone,
// for a limited time (DESIGN §6). X and Y are its cell.
type Pickup struct {
	X, Y    int
	Showing bool
	Ticks   int // ticks left of showing

	after int  // jewels that must be gone before it appears
	show  int  // how long it will show
	done  bool // it has been and gone, or never comes
}

// newPickup schedules a pickup for when a random share of a level's jewels,
// between two percentages, are gone: always at least one.
func newPickup(rng *rand.Rand, jewels, minPercent, maxPercent, show int) Pickup {
	percent := minPercent + rng.IntN(maxPercent-minPercent+1)
	return Pickup{after: max(jewels*percent/100, 1), show: show}
}

// lose takes away a pickup that is showing, for good. One still to come is left
// to appear on time.
func (p *Pickup) lose() {
	if p.Showing {
		p.Showing, p.done = false, true
	}
}

// World is one level in progress.
type World struct {
	// Changed holds the kobolds that changed form this tick, for the scene to show.
	Changed []Kobold

	Tuning config.Tuning

	Dwarf       Dwarf
	Kobolds     []Kobold
	Sacks       []Sack
	Gold        []Gold
	Shot        *Shot // nil when none is in flight
	Recharge    int   // ticks until the dwarf can fire again
	JewelsLeft  int
	JewelsTaken int
	Spawned     int // kobolds that have entered, less any sent back by Respawn
	Killed      int
	Caught      bool // the dwarf was caught; nothing moves until Respawn
	Over        bool // caught with no lives left
	Mushroom    Pickup
	Tankard     Pickup
	Chase       int // ticks left of the chase, when kobolds flee and can be eaten

	level        *Level
	session      *session.Session
	rng          *rand.Rand
	earth        []bool
	jewels       []bool
	jewelTotal   int // the jewels the level began with
	spawnWait    int
	pressed      Dir // the direction pushed this tick, for holding up a sack
	cleared      bool
	jewelRun     int // jewels taken in the current run
	jewelRunWait int // ticks left to take the next one
	eatPoints    int // what the next kobold eaten scores
	events       []Event
	dist         []int // scratch: tunnel distance from the dwarf
	queue        []int // scratch: the breadth-first search's queue
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
		spawnWait: t.KoboldSpawnTicks,
		dist:      make([]int, len(l.earth)),
		Tankard:   Pickup{done: true},
	}
	w.placeDwarf()
	for _, s := range l.sacks {
		w.Sacks = append(w.Sacks, Sack{X: s[0] * config.CellSize, Y: s[1] * config.CellSize})
	}
	for _, j := range w.jewels {
		if j {
			w.JewelsLeft++
		}
	}
	w.jewelTotal = w.JewelsLeft
	w.Mushroom = newPickup(rng, w.jewelTotal, config.MushroomGoneMinPercent, config.MushroomGoneMaxPercent, config.MushroomShowTicks)
	if t.TankardOneIn > 0 && rng.IntN(t.TankardOneIn) == 0 {
		w.Tankard = newPickup(rng, w.jewelTotal, config.TankardGoneMinPercent, config.TankardGoneMaxPercent, config.TankardShowTicks)
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
// kobold destroyed (DESIGN D5).
func (w *World) Cleared() bool { return w.cleared }

// Update advances one tick with dir pushed and fire pressed, and returns what
// happened. The slice is valid until the next Update.
func (w *World) Update(dir Dir, fire bool) []Event {
	w.events = w.events[:0]
	w.Changed = w.Changed[:0]
	if w.cleared || w.Caught {
		return w.events
	}
	w.jewelRunWait = max(w.jewelRunWait-1, 0)
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
	w.updateChase()
	if w.updatePickup(&w.Tankard, TankardAppeared) {
		w.session.Lives++
		w.events = append(w.events, ExtraLife)
	}
	for i := range w.Kobolds {
		w.moveKobold(&w.Kobolds[i])
	}
	w.shotHits()
	w.moveSacks()
	w.updateGold()
	if w.checkCleared() || w.Caught {
		return w.events
	}
	w.checkCaught()
	return w.events
}

// checkCleared finishes the level if its goal has been met.
func (w *World) checkCleared() bool {
	allShot := w.Tuning.KoboldTotal > 0 && w.Killed >= w.Tuning.KoboldTotal
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
	w.pressed = dir
	if centred(d.X, d.Y) {
		d.Moving, d.Pushing, d.Digging = None, false, false
		if dir != None {
			d.face(dir)
			if x, y := d.X/config.CellSize, d.Y/config.CellSize; inField(x, y, dir) {
				if enters, pushing := w.meetSack(x, y, dir); enters {
					d.Moving, d.Pushing = dir, pushing
					dx, dy := dir.delta()
					d.Digging = w.Earth(x+dx, y+dy)
				}
			}
		}
	} else if dir != None && dir == d.Moving.opposite() {
		d.face(dir)
		d.Moving = dir
	}
	speed := config.DwarfSpeed
	if d.Pushing {
		speed = w.pusherSpeed(d.X, d.Y, d.Moving)
	}
	dx, dy := d.Moving.delta()
	d.X += dx * speed
	d.Y += dy * speed
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
		w.countJewelRun()
	}
}

// countJewelRun extends the run of jewels, or starts a new one if the last was
// too long ago, and scores a run once it is long enough (DESIGN §5).
func (w *World) countJewelRun() {
	if w.jewelRunWait == 0 {
		w.jewelRun = 0
	}
	w.jewelRun++
	w.jewelRunWait = config.JewelRunTicks
	if w.jewelRun == config.JewelRunCount {
		w.jewelRun = 0
		w.addScore(config.JewelRunPoints)
		w.events = append(w.events, JewelRun)
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
// on the first kobold it meets.
func (w *World) moveShot() {
	s := w.Shot
	if s == nil {
		return
	}
	dx, dy := s.Dir.delta()
	s.X += dx * config.ShotSpeed
	s.Y += dy * config.ShotSpeed
	if s.X < 0 || s.Y < 0 || s.X >= config.FieldCols*config.CellSize || s.Y >= config.FieldRows*config.CellSize ||
		w.Earth(s.X/config.CellSize, s.Y/config.CellSize) || w.stopsShot(s.X, s.Y) {
		w.Shot = nil
		return
	}
	w.shotHits()
}

// shotHits ends the shot on any kobold it now touches. It is also run after
// the kobolds move, so that one can't step through the shot.
func (w *World) shotHits() {
	s := w.Shot
	if s == nil {
		return
	}
	for i := range w.Kobolds {
		g := &w.Kobolds[i]
		if s.X >= g.X && s.X < g.X+config.CellSize && s.Y >= g.Y && s.Y < g.Y+config.CellSize {
			w.Kobolds = append(w.Kobolds[:i], w.Kobolds[i+1:]...)
			w.Shot = nil
			w.killed()
			w.addScore(config.KoboldPoints)
			w.events = append(w.events, KoboldShot)
			return
		}
	}
}

// spawn brings in the next kobold when it is due, room is left on the field,
// and the dwarf is not standing by the entrance.
func (w *World) spawn() {
	t := &w.Tuning
	if w.Chase > 0 || w.Spawned >= t.KoboldTotal || len(w.Kobolds) >= t.KoboldsAtOnce {
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
	w.Kobolds = append(w.Kobolds, Kobold{X: w.level.spawnX * config.CellSize, Y: w.level.spawnY * config.CellSize})
	w.Spawned++
	w.spawnWait = t.KoboldSpawnTicks
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// moveKobold moves a kobold, choosing its way each time it is centred on a
// cell. A cell of earth it reaches, as a goblin or just after being one, is
// dug out, which can bring down a sack above it. A walker pushes a sack it
// walks into sideways; a goblin destroys any sack it touches, and eats any
// jewel it reaches (DESIGN §6). Any kobold takes gold it reaches. No one scores
// for these.
func (w *World) moveKobold(g *Kobold) {
	if g.Goblin > 0 {
		if g.Goblin--; g.Goblin == 0 {
			g.impatience = 0
			w.Changed = append(w.Changed, *g)
		}
	}
	g.stuck = false
	steps := w.Tuning.KoboldSpeed
	if e := w.Tuning.KoboldExtraOneIn; e > 0 && steps > 0 && g.Goblin == 0 && w.rng.IntN(e) == 0 {
		steps++
	}
	// A pixel at a time, so that no step passes over the centre of a cell.
	for i := 0; i < max(steps, 1); i++ {
		w.stepKobold(g, min(steps, 1))
		if g.Pushing {
			return
		}
	}
	w.growImpatient(g)
}

// growImpatient turns a kobold into a goblin once it has crossed other
// kobolds, or been stuck on a sack, for long enough (DESIGN D8).
func (w *World) growImpatient(g *Kobold) {
	if g.Goblin > 0 || w.Tuning.GoblinAfterTicks < 0 {
		return
	}
	if g.stuck || w.crossing(g) {
		g.impatience++
	}
	if g.impatience > w.Tuning.GoblinAfterTicks {
		g.Goblin, g.impatience = w.Tuning.GoblinTicks, 0
		w.Changed = append(w.Changed, *g)
	}
}

// crossing reports whether a kobold overlaps another.
func (w *World) crossing(g *Kobold) bool {
	reach := config.CellSize - 2*config.ContactInset
	for i := range w.Kobolds {
		if o := &w.Kobolds[i]; o != g && abs(o.X-g.X) < reach && abs(o.Y-g.Y) < reach {
			return true
		}
	}
	return false
}

// stepKobold moves a kobold the given pixels, at most one.
func (w *World) stepKobold(g *Kobold, speed int) {
	if centred(g.X, g.Y) {
		g.Pushing = false
		g.Moving = w.chooseWay(g)
		switch g.Moving {
		case Left:
			g.FacesLeft = true
		case Right:
			g.FacesLeft = false
		}
		if g.Goblin == 0 && g.Moving != None {
			enters, pushing := w.meetSack(g.X/config.CellSize, g.Y/config.CellSize, g.Moving)
			if !enters {
				g.Moving, g.stuck = None, true
			}
			g.Pushing = pushing
		}
	}
	if g.Pushing {
		speed = w.pusherSpeed(g.X, g.Y, g.Moving)
		g.stuck = g.stuck || speed == 0
	}
	dx, dy := g.Moving.delta()
	g.X += dx * speed
	g.Y += dy * speed
	cx, cy := (g.X+config.CellSize/2)/config.CellSize, (g.Y+config.CellSize/2)/config.CellSize
	w.earth[cy*config.FieldCols+cx] = false
	if g.Goblin > 0 {
		w.Sacks = slices.DeleteFunc(w.Sacks, func(s Sack) bool {
			return abs(s.X-g.X) < config.CellSize && abs(s.Y-g.Y) < config.CellSize
		})
		if w.Jewel(cx, cy) {
			w.jewels[cy*config.FieldCols+cx] = false
			w.JewelsLeft--
		}
	}
	w.Gold = slices.DeleteFunc(w.Gold, func(gold Gold) bool { return gold.X == cx && gold.Y == cy })
}

// chooseWay takes a shortest way to the dwarf, or in a chase the way furthest
// from it, through the tunnels or, for a goblin, through earth too, with ties
// broken at random. With no way there, the kobold wanders, turning back only
// at a dead end.
func (w *World) chooseWay(g *Kobold) Dir {
	digs := g.Goblin > 0
	w.measure(digs)
	x, y := g.X/config.CellSize, g.Y/config.CellSize
	best, ways, choice := -1, 0, None
	back, canBack := g.Moving.opposite(), false
	for _, dir := range dirs {
		if !w.walkable(x, y, dir, digs) {
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
		case best < 0 || (w.Chase == 0 && dist < best) || (w.Chase > 0 && dist > best):
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
		if dir != back && w.walkable(x, y, dir, digs) {
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

// walkable reports whether a kobold can step from cell (x, y) in direction
// dir: kobolds keep to the tunnels unless they dig. A walker can't go up or
// down through a sack, but may try to push one sideways, and a goblin goes
// where it likes.
func (w *World) walkable(x, y int, dir Dir, digs bool) bool {
	if !inField(x, y, dir) {
		return false
	}
	if digs {
		return true
	}
	dx, dy := dir.delta()
	if (dir == Up || dir == Down) && w.sackAt(x+dx, y+dy) {
		return false
	}
	return !w.Earth(x+dx, y+dy)
}

// measure fills w.dist with each cell's distance from the dwarf, or -1 if
// there is no way.
func (w *World) measure(digs bool) {
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
			if !w.walkable(cx, cy, dir, digs) {
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

// updatePickup watches for a pickup's jewels to go, runs its time showing, and
// reports whether the dwarf has just collected it. If no cell is open when they
// have gone, it tries again each tick.
func (w *World) updatePickup(p *Pickup, appeared Event) bool {
	if p.done {
		return false
	}
	if !p.Showing {
		if w.jewelTotal-w.JewelsLeft < p.after {
			return false
		}
		x, y, ok := w.pickCell()
		if !ok {
			return false
		}
		p.X, p.Y, p.Showing, p.Ticks = x, y, true, p.show
		w.events = append(w.events, appeared)
	}
	if x, y := w.Dwarf.cell(); x == p.X && y == p.Y {
		p.Showing, p.done = false, true
		return true
	}
	if p.Ticks--; p.Ticks <= 0 {
		p.Showing, p.done = false, true
	}
	return false
}

// pickCell chooses a random open cell for a pickup (DESIGN §6). It looks at the
// cells in a fixed order, so that a seed always gives the same choice.
func (w *World) pickCell() (x, y int, ok bool) {
	open := 0
	for y := range config.FieldRows {
		for x := range config.FieldCols {
			if w.pickupFits(x, y) {
				open++
			}
		}
	}
	if open == 0 {
		return 0, 0, false
	}
	choice := w.rng.IntN(open)
	for y := range config.FieldRows {
		for x := range config.FieldCols {
			if w.pickupFits(x, y) {
				if choice == 0 {
					return x, y, true
				}
				choice--
			}
		}
	}
	return 0, 0, false
}

// pickupFits reports whether cell (x, y) is open: clear of earth, a jewel, a
// sack, gold, a kobold, another pickup and the kobolds' entrance, and far enough
// from the dwarf.
func (w *World) pickupFits(x, y int) bool {
	dx, dy := w.Dwarf.cell()
	if w.Earth(x, y) || w.Jewel(x, y) || abs(x-dx)+abs(y-dy) < config.PickupMinCells ||
		(x == w.level.spawnX && y == w.level.spawnY) ||
		(w.Mushroom.Showing && x == w.Mushroom.X && y == w.Mushroom.Y) ||
		(w.Tankard.Showing && x == w.Tankard.X && y == w.Tankard.Y) {
		return false
	}
	touches := func(px, py int) bool {
		return abs(px-x*config.CellSize) < config.CellSize && abs(py-y*config.CellSize) < config.CellSize
	}
	for _, s := range w.Sacks {
		if touches(s.X, s.Y) {
			return false
		}
	}
	for _, g := range w.Kobolds {
		if touches(g.X, g.Y) {
			return false
		}
	}
	for _, g := range w.Gold {
		if g.X == x && g.Y == y {
			return false
		}
	}
	return true
}

// updateChase runs the level's one mushroom, whose collecting starts the chase,
// and runs the chase down (DESIGN §6).
func (w *World) updateChase() {
	if w.updatePickup(&w.Mushroom, MushroomAppeared) {
		w.Chase, w.eatPoints = w.Tuning.ChaseTicks, config.EatPoints
		w.addScore(config.MushroomPoints)
		w.events = append(w.events, MushroomTaken)
		return
	}
	if w.Chase > 0 {
		if w.Chase--; w.Chase == 0 {
			w.events = append(w.events, ChaseOver)
		}
	}
}

// checkCaught costs the dwarf a life if a kobold is touching it, or in a chase
// eats the kobold, for a score that doubles each time. Boxes are inset so that
// only a real overlap counts.
func (w *World) checkCaught() {
	reach := config.CellSize - 2*config.ContactInset
	for i := len(w.Kobolds) - 1; i >= 0; i-- {
		g := &w.Kobolds[i]
		if abs(g.X-w.Dwarf.X) >= reach || abs(g.Y-w.Dwarf.Y) >= reach {
			continue
		}
		if w.Chase == 0 {
			w.catchDwarf()
			return
		}
		w.Kobolds = append(w.Kobolds[:i], w.Kobolds[i+1:]...)
		w.killed()
		w.addScore(w.eatPoints)
		w.eatPoints *= 2
		w.events = append(w.events, KoboldEaten)
	}
}

// killed counts a kobold destroyed. One destroyed in a chase is replaced by
// another to come, so a chase can't clear the level (DESIGN §6).
func (w *World) killed() {
	w.Killed++
	if w.Chase > 0 {
		w.Tuning.KoboldTotal++
	}
}

func (w *World) catchDwarf() {
	w.Caught = true
	if w.Chase > 0 {
		w.Chase = 0
		w.events = append(w.events, ChaseOver)
	}
	w.Over = w.session.LoseLife()
	w.events = append(w.events, DwarfCaught)
}

// sackAt reports whether a sack, not falling, fills cell (x, y). A sliding
// sack fills both cells it spans. A falling sack blocks nothing: it crushes
// what it meets instead.
func (w *World) sackAt(x, y int) bool {
	for i := range w.Sacks {
		s := &w.Sacks[i]
		if !s.gone && s.State != SackFalling && s.Y == y*config.CellSize &&
			(s.X/config.CellSize == x || (s.X+config.CellSize-1)/config.CellSize == x) {
			return true
		}
	}
	return false
}

// stopsShot reports whether a sack, falling or not, or gold is at the point
// (x, y).
func (w *World) stopsShot(x, y int) bool {
	inside := func(left, top int) bool {
		return x >= left && x < left+config.CellSize && y >= top && y < top+config.CellSize
	}
	for i := range w.Sacks {
		if inside(w.Sacks[i].X, w.Sacks[i].Y) {
			return true
		}
	}
	for _, g := range w.Gold {
		if inside(g.X*config.CellSize, g.Y*config.CellSize) {
			return true
		}
	}
	return false
}

// meetSack reports whether a walker, centred on cell (x, y), can go in
// direction dir given any sack in the next cell, and whether it pushes one to
// do so. Nothing can go up or down through a sack (DESIGN §6).
func (w *World) meetSack(x, y int, dir Dir) (enters, pushing bool) {
	dx, dy := dir.delta()
	if !w.sackAt(x+dx, y+dy) {
		return true, false
	}
	if dir != Left && dir != Right {
		return false, false
	}
	left := (x + dx) * config.CellSize
	for i := range w.Sacks {
		s := &w.Sacks[i]
		if s.State != SackFalling && s.Y == y*config.CellSize && abs(s.X-left) < config.CellSize {
			if !w.pushSack(s, dir) {
				return false, false
			}
		}
	}
	return true, true
}

// pusherSpeed is how far something pushing a sack moves this tick: at the
// sack's speed, unless the sack, stopped ahead of it, would be overlapped.
func (w *World) pusherSpeed(x, y int, dir Dir) int {
	dx, _ := dir.delta()
	for i := range w.Sacks {
		s := &w.Sacks[i]
		if s.State != SackFalling && abs(s.Y-y) < config.CellSize &&
			(s.X-x)*dx > 0 && abs(s.X-x) < config.CellSize {
			return 0
		}
	}
	return config.SackSlideSpeed
}

// blocksSack reports whether something at (x, y) is in the way of a sack in
// row pixel sy moving to column pixel nx in the direction whose sign is dx.
func blocksSack(x, y, nx, sy, dx int) bool {
	return abs(y-sy) < config.CellSize && (x-nx)*dx >= 0 && abs(x-nx) < config.CellSize
}

// pushSack starts or keeps sack s sliding in direction dir, for its next step,
// and reports whether it can go (DESIGN §6). The edge of the field, the dwarf
// or a kobold stops it. Earth and jewels don't, and it pushes other sacks
// ahead of it unless they can't go. A falling sack ignores the push. A sack
// stopped on a cell comes to rest, and falls at once with nothing under it.
func (w *World) pushSack(s *Sack, dir Dir) bool {
	if s.State == SackFalling {
		return true
	}
	if (s.X == 0 && dir == Left) || (s.X == (config.FieldCols-1)*config.CellSize && dir == Right) {
		return false
	}
	dx, _ := dir.delta()
	nx := s.X + dx*config.SackSlideSpeed
	can := !blocksSack(w.Dwarf.X, w.Dwarf.Y, nx, s.Y, dx)
	for i := range w.Kobolds {
		can = can && !blocksSack(w.Kobolds[i].X, w.Kobolds[i].Y, nx, s.Y, dx)
	}
	for i := range w.Sacks {
		o := &w.Sacks[i]
		if can && o != s && !o.gone && o.State != SackFalling &&
			blocksSack(o.X, o.Y, nx, s.Y, dx) && !w.pushSack(o, dir) {
			can = false
		}
	}
	if !can {
		if s.X%config.CellSize == 0 {
			w.settle(s)
		}
		return false
	}
	if s.State != SackSliding {
		w.events = append(w.events, SackPushed)
	}
	s.State, s.Slide, s.Wobble = SackSliding, dir, 0
	return true
}

// settle ends a sideways move on a cell. With nothing under it, a sack falls
// at once, without wobbling.
func (w *World) settle(s *Sack) {
	s.Slide = None
	if w.supported(s) {
		s.State, s.Wobble = SackResting, 0
		return
	}
	w.startFall(s)
}

func (w *World) startFall(s *Sack) {
	s.State, s.Wobble = SackFalling, 0
	w.earth[s.Y/config.CellSize*config.FieldCols+s.X/config.CellSize] = false
}

// supported reports whether something holds up a sack centred on a cell: solid
// earth or the bottom of the field. Another sack doesn't: only the cell counts.
func (w *World) supported(s *Sack) bool {
	x, y := s.X/config.CellSize, s.Y/config.CellSize
	return y+1 >= config.FieldRows || w.Earth(x, y+1)
}

// holdsUp reports whether the dwarf, trying to move up or down in the sack's
// column, in its row or just under it, keeps it from starting to wobble.
func (w *World) holdsUp(s *Sack) bool {
	d := &w.Dwarf
	if w.pressed != Up && w.pressed != Down && d.Moving != Up && d.Moving != Down {
		return false
	}
	row, sackRow := d.Y/config.CellSize, s.Y/config.CellSize
	return d.X/config.CellSize == s.X/config.CellSize && (row == sackRow || row == sackRow+1)
}

// moveSacks runs the sacks (DESIGN §6, D6). A sack with nothing under it
// wobbles as a warning, then falls, crushing what it meets. One that lands
// after falling more than a cell bursts into gold.
func (w *World) moveSacks() {
	for i := range w.Sacks {
		s := &w.Sacks[i]
		if s.gone {
			continue
		}
		switch s.State {
		case SackResting:
			if !w.supported(s) && !w.holdsUp(s) {
				s.State, s.Wobble = SackWobble, config.SackWobbleTicks
				w.events = append(w.events, SackWobbling)
			}
		case SackWobble:
			s.Wobble--
			if s.Wobble <= 0 {
				w.startFall(s)
			}
		case SackSliding:
			w.slide(s)
		case SackFalling:
			w.fall(s)
		}
	}
	w.Sacks = slices.DeleteFunc(w.Sacks, func(s Sack) bool { return s.gone })
}

// slide moves a sack one step. If something has got in its way it turns back,
// which takes it to the cell it came from. It removes any gold it reaches.
func (w *World) slide(s *Sack) {
	if !w.pushSack(s, s.Slide) {
		if s.State == SackSliding {
			s.Slide = s.Slide.opposite()
		}
		return
	}
	dx, _ := s.Slide.delta()
	s.X += dx * config.SackSlideSpeed
	w.Gold = slices.DeleteFunc(w.Gold, func(g Gold) bool {
		return g.Y*config.CellSize == s.Y && abs(g.X*config.CellSize-s.X) < config.CellSize
	})
	if s.X%config.CellSize == 0 {
		w.settle(s)
	}
}

// fall moves a falling sack one step. It destroys any jewel in the cell it is
// entering, turns back kobolds climbing in its column, and goes through other
// sacks, destroying those and any gold it is touching where it lands.
func (w *World) fall(s *Sack) {
	s.Y += config.SackFallSpeed
	s.Fallen += config.SackFallSpeed
	x, y := s.X/config.CellSize, (s.Y+config.CellSize-1)/config.CellSize
	w.earth[y*config.FieldCols+x] = false
	if w.Jewel(x, y) {
		w.jewels[y*config.FieldCols+x] = false
		w.JewelsLeft--
	}
	for i := range w.Kobolds {
		if g := &w.Kobolds[i]; g.X/config.CellSize == x && g.Moving == Up {
			g.Moving = Down
		}
	}
	w.crush(s)
	if !centred(s.X, s.Y) || !w.supported(s) {
		return
	}
	for i := range w.Sacks {
		if o := &w.Sacks[i]; o != s && abs(o.X-s.X) < config.CellSize && abs(o.Y-s.Y) < config.CellSize {
			o.gone = true
		}
	}
	w.Gold = slices.DeleteFunc(w.Gold, func(g Gold) bool {
		return abs(g.X*config.CellSize-s.X) < config.CellSize && abs(g.Y*config.CellSize-s.Y) < config.CellSize
	})
	if s.Fallen > config.CellSize {
		w.Gold = append(w.Gold, Gold{X: s.X / config.CellSize, Y: s.Y / config.CellSize, Ticks: config.GoldTicks})
		w.events = append(w.events, SackBurst)
		s.gone = true
		return
	}
	s.State, s.Fallen = SackResting, 0
	w.events = append(w.events, SackLanded)
}

// crush destroys the kobolds, and catches the dwarf, under a falling sack.
// Anything above the sack's top is safe.
func (w *World) crush(s *Sack) {
	reach := config.CellSize - 2*config.ContactInset
	touches := func(x, y int) bool {
		return abs(x-s.X) < reach && abs(y-s.Y) < reach && y >= s.Y
	}
	for i := len(w.Kobolds) - 1; i >= 0; i-- {
		if g := &w.Kobolds[i]; touches(g.X, g.Y) {
			w.Kobolds = append(w.Kobolds[:i], w.Kobolds[i+1:]...)
			w.Killed++
			w.addScore(config.KoboldPoints)
			w.events = append(w.events, KoboldCrushed)
		}
	}
	if !w.Caught && touches(w.Dwarf.X, w.Dwarf.Y) {
		w.catchDwarf()
	}
}

// updateGold ages the gold and lets the dwarf collect what it is standing on.
// Gold with the cell under it dug out soon goes.
func (w *World) updateGold() {
	x, y := w.Dwarf.cell()
	kept := w.Gold[:0]
	for _, g := range w.Gold {
		if g.Y+1 < config.FieldRows && !w.Earth(g.X, g.Y+1) {
			g.Ticks = min(g.Ticks, config.GoldDugUnderTicks)
		}
		g.Ticks--
		switch {
		case g.X == x && g.Y == y:
			w.addScore(config.GoldPoints)
			w.events = append(w.events, GoldTaken)
		case g.Ticks > 0:
			kept = append(kept, g)
		}
	}
	w.Gold = kept
}

// Respawn puts the dwarf back at its start after being caught. What it dug
// and took stays as it is. The kobolds on the field go back to wait their turn.
// All gold goes, as does any sack on the move, wobbling, or on the dwarf's start.
func (w *World) Respawn() {
	w.Gold = w.Gold[:0]
	w.Sacks = slices.DeleteFunc(w.Sacks, func(s Sack) bool {
		return s.State != SackResting || (s.X == w.level.startX*config.CellSize && s.Y == w.level.startY*config.CellSize)
	})
	w.Spawned -= len(w.Kobolds)
	w.Kobolds = w.Kobolds[:0]
	w.Shot = nil
	w.Recharge = 0
	w.Caught = false
	w.jewelRun, w.jewelRunWait = 0, 0
	w.Mushroom.lose()
	w.Tankard.lose()
	w.spawnWait = w.Tuning.KoboldSpawnTicks
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
