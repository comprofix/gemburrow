package world

import (
	"slices"
	"testing"

	"github.com/comprofix/gemburrow/internal/config"
)

// corridor opens row y from column x0 to x1, with the dwarf and spawn cells
// left for the caller to add.
func corridor(cells map[[2]int]byte, y, x0, x1 int) map[[2]int]byte {
	for x := x0; x <= x1; x++ {
		if _, ok := cells[[2]int{x, y}]; !ok {
			cells[[2]int{x, y}] = config.LevelTunnel
		}
	}
	return cells
}

func TestWalkingGoblinStepsExtraAndStaysOnTheGrid(t *testing.T) {
	w, _ := newWorld(t, corridor(map[[2]int]byte{{0, 3}: config.LevelDwarf}, 3, 1, 12))
	w.Tuning.DiggerEveryTicks, w.Tuning.GoblinSpawnTicks = 0, 1000*config.TPS
	w.Tuning.GoblinExtraOneIn = 2
	addGoblin(w, 12, 3)
	ticks := 0
	for w.Goblins[0].X > 6*cell {
		w.Update(None, false)
		ticks++
		if w.Goblins[0].Y != 3*cell {
			t.Fatalf("left its row: %+v", w.Goblins[0])
		}
	}
	if ticks >= 6*cell {
		t.Errorf("%d ticks for 6 cells, no faster than the base speed", ticks)
	}
}

// addGoblin puts a goblin on cell (x, y), counted as having entered.
func addGoblin(w *World, x, y int) {
	w.Goblins = append(w.Goblins, Goblin{X: x * cell, Y: y * cell})
	w.Spawned++
}

func TestGoblinsEnterInTurnUpToTheLimits(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	w.Tuning.GoblinsAtOnce = 2
	w.Tuning.GoblinTotal = 3
	wait := w.Tuning.GoblinSpawnTicks

	run(w, None, wait)
	if len(w.Goblins) != 0 {
		t.Errorf("%d goblins before the first was due", len(w.Goblins))
	}
	run(w, None, 1)
	if len(w.Goblins) != 1 {
		t.Fatalf("%d goblins when the first was due", len(w.Goblins))
	}
	run(w, None, 3*(wait+1))
	if len(w.Goblins) != 2 || w.Spawned != 2 {
		t.Errorf("%d on the field and %d entered, want the limit of 2", len(w.Goblins), w.Spawned)
	}
	w.Goblins = w.Goblins[:1]
	run(w, None, 3*(wait+1))
	if w.Spawned != 3 {
		t.Errorf("%d entered, want all 3", w.Spawned)
	}
	w.Goblins = w.Goblins[:0]
	run(w, None, 3*(wait+1))
	if w.Spawned != 3 || len(w.Goblins) != 0 {
		t.Errorf("%d entered past the total of 3", w.Spawned)
	}
}

func TestGoblinWaitsWhileTheDwarfIsAtTheEntrance(t *testing.T) {
	spawn := [2]int{config.FieldCols - 1, config.FieldRows - 2}
	w, _ := newWorld(t, map[[2]int]byte{{spawn[0] - 1, spawn[1]}: config.LevelDwarf})
	run(w, None, 2*w.Tuning.GoblinSpawnTicks)
	if len(w.Goblins) != 0 {
		t.Error("a goblin entered beside the dwarf")
	}
}

func TestGoblinChasesAlongTheTunnel(t *testing.T) {
	w, _ := newWorld(t, corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 0, 1, 5))
	addGoblin(w, 5, 0)
	run(w, None, cell/w.Tuning.GoblinSpeed)
	if g := w.Goblins[0]; g.X != 4*cell || g.Y != 0 || !g.FacesLeft {
		t.Errorf("goblin at (%d, %d), facing left %v", g.X, g.Y, g.FacesLeft)
	}
}

func TestGoblinTakesTheShortWay(t *testing.T) {
	// Two ways from (3, 2) to the dwarf at (0, 2): straight along the row, or
	// a long way round by the row above.
	cells := corridor(map[[2]int]byte{{0, 2}: config.LevelDwarf}, 2, 1, 3)
	cells = corridor(cells, 1, 0, 3)
	w, _ := newWorld(t, cells)
	addGoblin(w, 3, 2)
	run(w, None, 1)
	if m := w.Goblins[0].Moving; m != Left {
		t.Errorf("going %v, want left", m)
	}
}

func TestGoblinNeverEntersEarth(t *testing.T) {
	// The dwarf is walled off, so the goblin wanders its short tunnel.
	cells := corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 3, 5, 8)
	w, _ := newWorld(t, cells)
	w.Tuning.GoblinTotal = 1
	w.Tuning.DiggerEveryTicks = 0
	addGoblin(w, 6, 3)
	moved := false
	for range 500 {
		w.Update(None, false)
		g := w.Goblins[0]
		if g.Y != 3*cell || g.X < 5*cell || g.X > 8*cell {
			t.Fatalf("goblin left its tunnel: (%d, %d)", g.X, g.Y)
		}
		moved = moved || g.X != 6*cell
	}
	if !moved {
		t.Error("the goblin never moved")
	}
}

func TestGoblinFollowsNewTunnels(t *testing.T) {
	cells := corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 2, 0, 3)
	w, _ := newWorld(t, cells)
	addGoblin(w, 3, 2)
	// Digging down joins the dwarf's cell to the goblin's tunnel.
	run(w, Down, 2*ticksPerCell-1)
	w.measure(false)
	if d := w.dist[2*config.FieldCols+3]; d < 0 {
		t.Errorf("no way through the dug tunnel: distance %d", d)
	}
}

func TestCaughtCostsALifeAndRespawnKeepsTheDigging(t *testing.T) {
	cells := map[[2]int]byte{{3, 3}: config.LevelDwarf}
	w, s := newWorld(t, cells)
	run(w, Right, ticksPerCell)
	addGoblin(w, 4, 3)
	if ev := run(w, None, 1); !slices.Contains(ev, DwarfCaught) {
		t.Fatalf("events %v, want DwarfCaught", ev)
	}
	if s.Lives != config.StartLives-1 || !w.Caught || w.Over {
		t.Errorf("lives %d, caught %v, over %v", s.Lives, w.Caught, w.Over)
	}
	x := w.Dwarf.X
	if ev := run(w, Right, 5); len(ev) != 0 || w.Dwarf.X != x {
		t.Errorf("moved while caught: %v", ev)
	}
	spawned := w.Spawned
	w.Respawn()
	if w.Caught || w.Dwarf.X != 3*cell || w.Dwarf.Y != 3*cell {
		t.Errorf("after respawn: caught %v at (%d, %d)", w.Caught, w.Dwarf.X, w.Dwarf.Y)
	}
	if len(w.Goblins) != 0 || w.Spawned != spawned-1 {
		t.Errorf("goblins %d, entered %d", len(w.Goblins), w.Spawned)
	}
	if w.Earth(4, 3) {
		t.Error("the tunnel filled in")
	}
}

func TestNearMissIsNotCaught(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{3, 3}: config.LevelDwarf})
	w.Goblins = []Goblin{{X: 3*cell + cell - 2*config.ContactInset, Y: 3 * cell}}
	w.checkCaught()
	if w.Caught {
		t.Error("caught by boxes that only touch inside the inset")
	}
}

func TestLastLifeEndsTheGame(t *testing.T) {
	w, s := newWorld(t, map[[2]int]byte{{3, 3}: config.LevelDwarf})
	s.Lives = 1
	addGoblin(w, 3, 3)
	run(w, None, 1)
	if !w.Over || s.Lives != 0 {
		t.Errorf("over %v, lives %d", w.Over, s.Lives)
	}
}

func TestShotFliesTheWayFacedAndStopsAtEarth(t *testing.T) {
	w, _ := newWorld(t, corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 0, 1, 3))
	if ev := w.Update(None, true); !slices.Contains(ev, ShotFired) {
		t.Fatalf("events %v, want ShotFired", ev)
	}
	if w.Shot == nil || w.Shot.Dir != Right {
		t.Fatalf("shot %+v", w.Shot)
	}
	x := w.Shot.X
	run(w, None, 3)
	if w.Shot.X != x+3*config.ShotSpeed || w.Shot.Y != cell/2 {
		t.Errorf("shot at (%d, %d)", w.Shot.X, w.Shot.Y)
	}
	run(w, None, 4*cell/config.ShotSpeed)
	if w.Shot != nil {
		t.Errorf("shot still flying at (%d, %d)", w.Shot.X, w.Shot.Y)
	}
	if !w.Earth(4, 0) {
		t.Error("the shot dug earth")
	}
}

func TestShotStopsAtTheFieldEdge(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	run(w, Up, 1) // face the edge
	w.Update(None, true)
	run(w, None, cell/config.ShotSpeed)
	if w.Shot != nil {
		t.Errorf("shot at (%d, %d)", w.Shot.X, w.Shot.Y)
	}
}

func TestShotMustRecharge(t *testing.T) {
	w, _ := newWorld(t, corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 0, 1, config.FieldCols-1))
	w.Update(None, true)
	wait := w.Tuning.ShotRechargeTicks
	if ev := run(w, None, 1); slices.Contains(ev, ShotFired) {
		t.Fatal("fired with nothing pressed")
	}
	for range wait - 2 {
		if ev := w.Update(None, true); slices.Contains(ev, ShotFired) {
			t.Fatalf("fired again with %d ticks to wait", w.Recharge)
		}
	}
	if ev := w.Update(None, true); !slices.Contains(ev, ShotFired) {
		t.Errorf("did not fire when recharged: %v", ev)
	}
}

func TestShotDestroysAGoblinAndScores(t *testing.T) {
	w, s := newWorld(t, corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 0, 1, 6))
	addGoblin(w, 6, 0)
	w.Update(None, true)
	ev := run(w, None, 5*cell/config.ShotSpeed)
	if !slices.Contains(ev, GoblinShot) {
		t.Fatalf("events %v, want GoblinShot", ev)
	}
	if len(w.Goblins) != 0 || w.Shot != nil || w.Killed != 1 {
		t.Errorf("goblins %d, shot %v, killed %d", len(w.Goblins), w.Shot, w.Killed)
	}
	if s.Score != config.GoblinPoints {
		t.Errorf("score %d", s.Score)
	}
}

func TestShootingEveryGoblinClearsTheLevel(t *testing.T) {
	w, _ := newWorld(t, corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 0, 1, 6))
	w.Tuning.GoblinTotal = 1
	addGoblin(w, 6, 0)
	w.Update(None, true)
	if ev := run(w, None, 5*cell/config.ShotSpeed); !slices.Contains(ev, LevelCleared) {
		t.Errorf("events %v, want LevelCleared", ev)
	}
	if !w.Cleared() {
		t.Error("not cleared")
	}
}

func TestNoGoblinsDoesNotClearTheLevel(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	w.Tuning.GoblinTotal = 0
	run(w, None, 5)
	if w.Cleared() {
		t.Error("cleared with jewels left")
	}
}

func TestExtraLifeOnPassingTheThreshold(t *testing.T) {
	w, s := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	s.Score = config.ExtraLifeEvery - 1
	lives := s.Lives
	ev := []Event{}
	w.events = w.events[:0]
	w.addScore(1)
	ev = append(ev, w.events...)
	if s.Lives != lives+1 || !slices.Contains(ev, ExtraLife) {
		t.Errorf("lives %d, events %v", s.Lives, ev)
	}
	w.addScore(config.JewelPoints)
	if s.Lives != lives+1 {
		t.Errorf("a second life with no new threshold: %d", s.Lives)
	}
}

func TestSameSeedSameChase(t *testing.T) {
	play := func() (int, int) {
		cells := corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 4, 0, config.FieldCols-1)
		w, _ := newWorld(t, cells)
		addGoblin(w, 9, 4)
		run(w, None, 400)
		return w.Goblins[0].X, w.Goblins[0].Y
	}
	ax, ay := play()
	bx, by := play()
	if ax != bx || ay != by {
		t.Errorf("(%d, %d) then (%d, %d)", ax, ay, bx, by)
	}
}
