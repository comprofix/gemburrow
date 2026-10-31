package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/gemburrow/internal/config"
	"github.com/comprofix/gemburrow/internal/session"
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

func TestWalkingKoboldStepsExtraAndStaysOnTheGrid(t *testing.T) {
	w, _ := newWorld(t, corridor(map[[2]int]byte{{0, 3}: config.LevelDwarf}, 3, 1, 12))
	w.Tuning.GoblinAfterTicks, w.Tuning.KoboldSpawnTicks = -1, 1000*config.TPS
	w.Tuning.KoboldExtraOneIn = 2
	addKobold(w, 12, 3)
	ticks := 0
	for w.Kobolds[0].X > 6*cell {
		w.Update(None, false)
		ticks++
		if w.Kobolds[0].Y != 3*cell {
			t.Fatalf("left its row: %+v", w.Kobolds[0])
		}
	}
	if ticks >= 6*cell {
		t.Errorf("%d ticks for 6 cells, no faster than the base speed", ticks)
	}
}

// addKobold puts a kobold on cell (x, y), counted as having entered.
func addKobold(w *World, x, y int) {
	w.Kobolds = append(w.Kobolds, Kobold{X: x * cell, Y: y * cell})
	w.Spawned++
}

func TestKoboldsEnterInTurnUpToTheLimits(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	w.Tuning.KoboldsAtOnce = 2
	w.Tuning.KoboldTotal = 3
	wait := w.Tuning.KoboldSpawnTicks

	run(w, None, wait)
	if len(w.Kobolds) != 0 {
		t.Errorf("%d kobolds before the first was due", len(w.Kobolds))
	}
	run(w, None, 1)
	if len(w.Kobolds) != 1 {
		t.Fatalf("%d kobolds when the first was due", len(w.Kobolds))
	}
	run(w, None, 3*(wait+1))
	if len(w.Kobolds) != 2 || w.Spawned != 2 {
		t.Errorf("%d on the field and %d entered, want the limit of 2", len(w.Kobolds), w.Spawned)
	}
	w.Kobolds = w.Kobolds[:1]
	run(w, None, 3*(wait+1))
	if w.Spawned != 3 {
		t.Errorf("%d entered, want all 3", w.Spawned)
	}
	w.Kobolds = w.Kobolds[:0]
	run(w, None, 3*(wait+1))
	if w.Spawned != 3 || len(w.Kobolds) != 0 {
		t.Errorf("%d entered past the total of 3", w.Spawned)
	}
}

func TestKoboldWaitsWhileTheDwarfIsAtTheEntrance(t *testing.T) {
	spawn := [2]int{config.FieldCols - 1, config.FieldRows - 2}
	w, _ := newWorld(t, map[[2]int]byte{{spawn[0] - 1, spawn[1]}: config.LevelDwarf})
	run(w, None, 2*w.Tuning.KoboldSpawnTicks)
	if len(w.Kobolds) != 0 {
		t.Error("a kobold entered beside the dwarf")
	}
}

func TestKoboldChasesAlongTheTunnel(t *testing.T) {
	w, _ := newWorld(t, corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 0, 1, 5))
	addKobold(w, 5, 0)
	run(w, None, cell/w.Tuning.KoboldSpeed)
	if g := w.Kobolds[0]; g.X != 4*cell || g.Y != 0 || !g.FacesLeft {
		t.Errorf("kobold at (%d, %d), facing left %v", g.X, g.Y, g.FacesLeft)
	}
}

func TestKoboldTakesTheShortWay(t *testing.T) {
	// Two ways from (3, 2) to the dwarf at (0, 2): straight along the row, or
	// a long way round by the row above.
	cells := corridor(map[[2]int]byte{{0, 2}: config.LevelDwarf}, 2, 1, 3)
	cells = corridor(cells, 1, 0, 3)
	w, _ := newWorld(t, cells)
	addKobold(w, 3, 2)
	run(w, None, 1)
	if m := w.Kobolds[0].Moving; m != Left {
		t.Errorf("going %v, want left", m)
	}
}

func TestKoboldNeverEntersEarth(t *testing.T) {
	// The dwarf is walled off, so the kobold wanders its short tunnel.
	cells := corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 3, 5, 8)
	w, _ := newWorld(t, cells)
	w.Tuning.KoboldTotal = 1
	w.Tuning.GoblinAfterTicks = -1
	addKobold(w, 6, 3)
	moved := false
	for range 500 {
		w.Update(None, false)
		g := w.Kobolds[0]
		if g.Y != 3*cell || g.X < 5*cell || g.X > 8*cell {
			t.Fatalf("kobold left its tunnel: (%d, %d)", g.X, g.Y)
		}
		moved = moved || g.X != 6*cell
	}
	if !moved {
		t.Error("the kobold never moved")
	}
}

func TestKoboldFollowsNewTunnels(t *testing.T) {
	cells := corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 2, 0, 3)
	w, _ := newWorld(t, cells)
	addKobold(w, 3, 2)
	// Digging down joins the dwarf's cell to the kobold's tunnel.
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
	addKobold(w, 4, 3)
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
	if len(w.Kobolds) != 0 || w.Spawned != spawned-1 {
		t.Errorf("kobolds %d, entered %d", len(w.Kobolds), w.Spawned)
	}
	if w.Earth(4, 3) {
		t.Error("the tunnel filled in")
	}
}

func TestNearMissIsNotCaught(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{3, 3}: config.LevelDwarf})
	w.Kobolds = []Kobold{{X: 3*cell + cell - 2*config.ContactInset, Y: 3 * cell}}
	w.checkCaught()
	if w.Caught {
		t.Error("caught by boxes that only touch inside the inset")
	}
}

func TestLastLifeEndsTheGame(t *testing.T) {
	w, s := newWorld(t, map[[2]int]byte{{3, 3}: config.LevelDwarf})
	s.Lives = 1
	addKobold(w, 3, 3)
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

func TestShotDestroysAKoboldAndScores(t *testing.T) {
	w, s := newWorld(t, corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 0, 1, 6))
	addKobold(w, 6, 0)
	w.Update(None, true)
	ev := run(w, None, 5*cell/config.ShotSpeed)
	if !slices.Contains(ev, KoboldShot) {
		t.Fatalf("events %v, want KoboldShot", ev)
	}
	if len(w.Kobolds) != 0 || w.Shot != nil || w.Killed != 1 {
		t.Errorf("kobolds %d, shot %v, killed %d", len(w.Kobolds), w.Shot, w.Killed)
	}
	if s.Score != config.KoboldPoints {
		t.Errorf("score %d", s.Score)
	}
}

func TestShootingEveryKoboldClearsTheLevel(t *testing.T) {
	w, _ := newWorld(t, corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 0, 1, 6))
	w.Tuning.KoboldTotal = 1
	addKobold(w, 6, 0)
	w.Update(None, true)
	if ev := run(w, None, 5*cell/config.ShotSpeed); !slices.Contains(ev, LevelCleared) {
		t.Errorf("events %v, want LevelCleared", ev)
	}
	if !w.Cleared() {
		t.Error("not cleared")
	}
}

func TestNoKoboldsDoesNotClearTheLevel(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	w.Tuning.KoboldTotal = 0
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
		addKobold(w, 9, 4)
		run(w, None, 400)
		return w.Kobolds[0].X, w.Kobolds[0].Y
	}
	ax, ay := play()
	bx, by := play()
	if ax != bx || ay != by {
		t.Errorf("(%d, %d) then (%d, %d)", ax, ay, bx, by)
	}
}

// chaseWorld is a level with one open row and its one kobold still to enter, the
// dwarf at the row's end, and no mushroom on its way.
func chaseWorld(t *testing.T) (*World, *session.Session) {
	t.Helper()
	cells := corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 0, 1, config.FieldCols-1)
	w, s := newWorld(t, cells)
	w.Tuning.GoblinAfterTicks, w.Tuning.KoboldSpawnTicks, w.spawnWait = -1, 0, 0
	w.Tuning.KoboldTotal = 1
	w.Mushroom.done = true
	return w, s
}

// quietWorld is a level with one open row, no kobolds to enter, and a mushroom
// due next tick.
func quietWorld(t *testing.T) *World {
	t.Helper()
	cells := corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 0, 1, config.FieldCols-1)
	w, _ := newWorld(t, cells)
	w.Tuning.KoboldTotal = 0
	w.Mushroom.wait = 0
	return w
}

func TestTheMushroomIsDueWithinItsDelays(t *testing.T) {
	for seed := range uint64(50) {
		w, _ := newWorld(t, corridor(map[[2]int]byte{{0, 0}: config.LevelDwarf}, 0, 1, config.FieldCols-1))
		w.rng = rand.New(rand.NewPCG(seed, 1))
		m := newPickup(w.rng, config.MushroomDelayMinTicks, config.MushroomDelayMaxTicks, config.MushroomShowTicks)
		if m.wait < config.MushroomDelayMinTicks || m.wait > config.MushroomDelayMaxTicks {
			t.Errorf("seed %d: wait %d", seed, m.wait)
		}
	}
}

// Every layout must show its mushroom with no kobold killed, however the dwarf
// fares, which is what the playtest to level 7 never saw (DESIGN D7).
func TestEveryLevelShowsAMushroomWithNoKills(t *testing.T) {
	for n := 1; n <= len(config.Levels); n++ {
		l, err := Parse(config.LevelFor(n))
		if err != nil {
			t.Fatal(err)
		}
		s := session.New(config.Skills[1], 1000)
		w := New(s, l, config.TuningFor(config.Skills[1], n), rand.New(rand.NewPCG(uint64(n), 1)))
		for range config.MushroomDelayMaxTicks + 1 {
			w.Update(None, false)
			if w.Mushroom.Showing || w.Cleared() {
				break
			}
			if w.Caught {
				w.Respawn()
			}
		}
		if !w.Mushroom.Showing {
			t.Errorf("level %d: no mushroom, wait %d, cleared %v", n, w.Mushroom.wait, w.Cleared())
		}
	}
}

func TestTheMushroomChoosesAnOpenCellAwayFromTheDwarf(t *testing.T) {
	for seed := range uint64(50) {
		w := quietWorld(t)
		w.rng = rand.New(rand.NewPCG(seed, 2))
		ev := run(w, None, 1)
		m := w.Mushroom
		if !m.Showing || !slices.Contains(ev, MushroomAppeared) {
			t.Fatalf("seed %d: mushroom %+v, events %v", seed, m, ev)
		}
		dx, dy := w.Dwarf.cell()
		if w.Earth(m.X, m.Y) || abs(m.X-dx)+abs(m.Y-dy) < config.PickupMinCells {
			t.Errorf("seed %d: at (%d, %d), dwarf (%d, %d)", seed, m.X, m.Y, dx, dy)
		}
	}
}

func TestAPickupAvoidsSacksKoboldsGoldAndTheEntrance(t *testing.T) {
	w := quietWorld(t)
	y := 0
	w.Sacks = []Sack{{X: 5 * cell, Y: y}}
	w.Kobolds = []Kobold{{X: 7 * cell, Y: y}}
	w.Gold = []Gold{{X: 9, Y: y}}
	for _, x := range []int{5, 7, 9, w.level.spawnX} {
		if ly := w.level.spawnY; x == w.level.spawnX && w.pickupFits(x, ly) {
			t.Errorf("fits the entrance (%d, %d)", x, ly)
		} else if x != w.level.spawnX && w.pickupFits(x, y) {
			t.Errorf("fits (%d, %d)", x, y)
		}
	}
	if !w.pickupFits(11, y) {
		t.Error("doesn't fit a clear cell")
	}
}

func TestAPickupWaitsForAnOpenCell(t *testing.T) {
	w := quietWorld(t)
	for i := range w.earth {
		w.earth[i] = true
	}
	run(w, None, 3)
	if w.Mushroom.Showing || w.Mushroom.done {
		t.Fatalf("no open cell: %+v", w.Mushroom)
	}
	w.earth[12] = false
	run(w, None, 1)
	if m := w.Mushroom; !m.Showing || m.X != 12 || m.Y != 0 {
		t.Errorf("once open: %+v", m)
	}
}

func TestTheMushroomVanishesOnTimeAndNeverReturns(t *testing.T) {
	w := quietWorld(t)
	run(w, None, config.MushroomShowTicks-1)
	if !w.Mushroom.Showing {
		t.Fatal("gone too soon")
	}
	run(w, None, 1)
	if w.Mushroom.Showing {
		t.Fatal("still showing after its time")
	}
	if ev := run(w, None, config.MushroomDelayMaxTicks); w.Mushroom.Showing || slices.Contains(ev, MushroomAppeared) {
		t.Errorf("came back: %+v", w.Mushroom)
	}
}

func TestADeathLosesAShowingMushroomButNotOneToCome(t *testing.T) {
	w := quietWorld(t)
	run(w, None, 1)
	w.catchDwarf()
	w.Respawn()
	if ev := run(w, None, config.MushroomDelayMaxTicks); w.Mushroom.Showing || slices.Contains(ev, MushroomAppeared) {
		t.Errorf("came back: %+v", w.Mushroom)
	}

	w = quietWorld(t)
	w.Mushroom.wait = 10
	run(w, None, 4)
	w.catchDwarf()
	w.Respawn()
	if w.Mushroom.wait != 6 {
		t.Errorf("wait %d after the death", w.Mushroom.wait)
	}
	run(w, None, 7)
	if !w.Mushroom.Showing {
		t.Errorf("didn't come: %+v", w.Mushroom)
	}
}

func TestTheMushroomIsTheSameForTheSameSeed(t *testing.T) {
	place := func() Pickup {
		w := quietWorld(t)
		w.rng = rand.New(rand.NewPCG(7, 7))
		run(w, None, 1)
		return w.Mushroom
	}
	if a, b := place(), place(); a != b {
		t.Errorf("%+v then %+v", a, b)
	}
}

func TestTakingTheMushroomScoresAndStartsTheChase(t *testing.T) {
	w, s := chaseWorld(t)
	w.Mushroom = Pickup{X: 3, Y: 0, Showing: true, Ticks: 100}
	w.Spawned = 1
	w.Dwarf.X, w.Dwarf.Y = 3*cell, 0
	ev := run(w, None, 1)
	if w.Mushroom.Showing || w.Chase != w.Tuning.ChaseTicks || s.Score != config.MushroomPoints || !slices.Contains(ev, MushroomTaken) {
		t.Errorf("mushroom %+v, chase %d, score %d, events %v", w.Mushroom, w.Chase, s.Score, ev)
	}
	ev = run(w, None, w.Tuning.ChaseTicks)
	if w.Chase != 0 || !slices.Contains(ev, ChaseOver) {
		t.Errorf("chase %d after its time, events %v", w.Chase, ev)
	}
}

func TestEatingKoboldsDoublesTheScoreAndTheyAreReplaced(t *testing.T) {
	w, s := chaseWorld(t)
	w.Tuning.KoboldTotal, w.Spawned = 10, 10
	w.Chase, w.eatPoints = 1000, config.EatPoints
	for range 3 {
		w.Kobolds = []Kobold{{X: w.Dwarf.X, Y: w.Dwarf.Y}}
		w.checkCaught()
		if w.Caught {
			t.Fatal("caught in a chase")
		}
	}
	if want := 200 + 400 + 800; s.Score != want || w.Killed != 3 || w.Tuning.KoboldTotal != 13 {
		t.Errorf("score %d, want %d; killed %d, total %d", s.Score, want, w.Killed, w.Tuning.KoboldTotal)
	}
}

func TestKoboldsFleeAndNoneEnterInAChase(t *testing.T) {
	w, _ := chaseWorld(t)
	w.Tuning.KoboldTotal = 5
	w.Spawned = 1
	addKobold(w, 6, 0)
	w.Chase = 1000
	run(w, None, cell)
	if g := w.Kobolds[0]; g.X <= 6*cell {
		t.Errorf("came nearer: %+v", g)
	}
	if len(w.Kobolds) != 1 || w.Spawned != 2 {
		t.Errorf("a kobold entered: %d, %d", len(w.Kobolds), w.Spawned)
	}
}

func TestADeathEndsTheChase(t *testing.T) {
	w, _ := chaseWorld(t)
	w.Chase = 1000
	w.catchDwarf()
	if w.Chase != 0 {
		t.Errorf("chase %d after dying", w.Chase)
	}
}

// The play scene plays a tune for each event, so the chase ending must come
// before the death for the funeral march to be the tune left playing.
func TestACrushDuringAChaseEndsTheChaseBeforeTheDeath(t *testing.T) {
	w, _ := chaseWorld(t)
	w.Chase = 1000
	w.catchDwarf()
	if len(w.events) != 2 || w.events[0] != ChaseOver || w.events[1] != DwarfCaught {
		t.Errorf("events %v", w.events)
	}
}

func TestChangedReportsEachChangeOfForm(t *testing.T) {
	w, _ := newWorld(t, corridor(map[[2]int]byte{{0, 3}: config.LevelDwarf}, 3, 1, 12))
	w.Tuning.GoblinAfterTicks, w.Tuning.KoboldSpawnTicks = 0, 1000*config.TPS
	w.Tuning.GoblinTicks = 3
	addKobold(w, 12, 3)
	changes := 0
	for range 200 {
		w.Update(None, false)
		changes += len(w.Changed)
		if changes >= 2 {
			return
		}
		w.Kobolds[0].impatience = 1 // keep it impatient, whatever it is doing
	}
	t.Errorf("%d changes of form", changes)
}
