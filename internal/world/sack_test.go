package world

import (
	"slices"
	"testing"

	"github.com/comprofix/gemburrow/internal/config"
)

func TestGoblinEatsAJewelWithoutScore(t *testing.T) {
	w, s := newWorld(t, map[[2]int]byte{{0, 5}: config.LevelDwarf, {3, 5}: config.LevelJewel})
	w.Tuning.GoblinAfterTicks, w.Tuning.KoboldSpawnTicks = -1, 1000*config.TPS
	addKobold(w, 4, 5)
	w.Kobolds[0].Goblin = 100
	jewels := w.JewelsLeft
	run(w, None, cell/w.Tuning.KoboldSpeed)
	if w.Jewel(3, 5) || w.JewelsLeft != jewels-1 || s.Score != 0 {
		t.Errorf("jewel %v, left %d of %d, score %d", w.Jewel(3, 5), w.JewelsLeft, jewels, s.Score)
	}
}

func TestWalkingKoboldLeavesAJewel(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 5}: config.LevelDwarf, {3, 5}: config.LevelJewel, {4, 5}: config.LevelTunnel})
	w.Tuning.GoblinAfterTicks, w.Tuning.KoboldSpawnTicks = -1, 1000*config.TPS
	addKobold(w, 4, 5)
	run(w, None, 2*cell/w.Tuning.KoboldSpeed)
	if !w.Jewel(3, 5) {
		t.Error("a walking kobold took the jewel")
	}
}

// wobbleAndFall is long enough for a sack to wobble and fall n cells.
func wobbleAndFall(n int) int {
	return config.SackWobbleTicks + 2 + n*cell/config.SackFallSpeed
}

func TestEverySackInTheLevelsStartsSupported(t *testing.T) {
	for i, rows := range config.Levels {
		l, err := Parse(rows)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range l.sacks {
			if s[1]+1 < config.FieldRows && !l.earth[(s[1]+1)*config.FieldCols+s[0]] {
				t.Errorf("level %d: the sack at %v has nothing under it", i+1, s)
			}
		}
	}
}

func TestSackRestsOnEarth(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf, {5, 3}: config.LevelSack})
	run(w, None, wobbleAndFall(3))
	if s := w.Sacks[0]; s.State != SackResting || s.Y != 3*cell {
		t.Errorf("sack %+v", s)
	}
}

func TestSackWobblesThenFallsOneCellAndRests(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 0}: config.LevelDwarf, {5, 3}: config.LevelSack, {5, 4}: config.LevelTunnel,
	})
	if ev := run(w, None, 1); !slices.Contains(ev, SackWobbling) {
		t.Fatalf("events %v, want SackWobbling", ev)
	}
	run(w, None, config.SackWobbleTicks-1)
	if s := w.Sacks[0]; s.Y != 3*cell {
		t.Errorf("fell while wobbling: %+v", s)
	}
	ev := run(w, None, wobbleAndFall(1))
	if !slices.Contains(ev, SackLanded) || slices.Contains(ev, SackBurst) {
		t.Errorf("events %v", ev)
	}
	if s := w.Sacks[0]; s.State != SackResting || s.Y != 4*cell {
		t.Errorf("sack %+v", s)
	}
	if w.Earth(5, 3) {
		t.Error("the sack's old cell is still earth")
	}
}

func TestSackBurstsAfterMoreThanACellAndGoldExpires(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 0}: config.LevelDwarf, {5, 3}: config.LevelSack,
		{5, 4}: config.LevelTunnel, {5, 5}: config.LevelTunnel,
	})
	if ev := run(w, None, wobbleAndFall(2)); !slices.Contains(ev, SackBurst) {
		t.Fatalf("events %v, want SackBurst", ev)
	}
	if len(w.Sacks) != 0 || len(w.Gold) != 1 || w.Gold[0].X != 5 || w.Gold[0].Y != 5 {
		t.Fatalf("sacks %v, gold %v", w.Sacks, w.Gold)
	}
	run(w, None, config.GoldTicks)
	if len(w.Gold) != 0 {
		t.Errorf("gold %v still here", w.Gold)
	}
}

func TestGoldScoresWhenCollected(t *testing.T) {
	w, s := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	w.Gold = []Gold{{X: 1, Y: 0, Ticks: config.GoldTicks}}
	if ev := run(w, Right, ticksPerCell); !slices.Contains(ev, GoldTaken) {
		t.Fatalf("events %v, want GoldTaken", ev)
	}
	if s.Score != config.GoldPoints || len(w.Gold) != 0 {
		t.Errorf("score %d, gold %v", s.Score, w.Gold)
	}
}

func TestDwarfPushesASackSideways(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf, {1, 0}: config.LevelSack})
	ev := run(w, Right, 2*ticksPerCell)
	if !slices.Contains(ev, SackPushed) {
		t.Fatalf("events %v, want SackPushed", ev)
	}
	if s := w.Sacks[0]; s.X != 2*cell || s.State != SackResting || w.Dwarf.X != cell {
		t.Errorf("sack %+v, dwarf x %d", s, w.Dwarf.X)
	}
}

func TestPushingAcrossAGapDropsTheSack(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 3}: config.LevelDwarf, {1, 3}: config.LevelSack,
		{2, 4}: config.LevelTunnel, {2, 5}: config.LevelTunnel,
	})
	ev := run(w, Right, 2*ticksPerCell)
	ev = append(ev, run(w, None, 2*cell/config.SackFallSpeed+1)...)
	if !slices.Contains(ev, SackBurst) || len(w.Gold) != 1 || w.Gold[0].X != 2 {
		t.Errorf("events %v, gold %v", ev, w.Gold)
	}
}

func TestSackCannotBePushed(t *testing.T) {
	for name, c := range map[string]struct {
		cells map[[2]int]byte
		dir   Dir
	}{
		"upwards":   {map[[2]int]byte{{1, 4}: config.LevelDwarf, {1, 3}: config.LevelSack}, Up},
		"downwards": {map[[2]int]byte{{1, 3}: config.LevelDwarf, {1, 4}: config.LevelSack}, Down},
		"past the edge": {map[[2]int]byte{
			{config.FieldCols - 2, 3}: config.LevelDwarf, {config.FieldCols - 1, 3}: config.LevelSack,
		}, Right},
		"with a sack against the edge": {map[[2]int]byte{
			{config.FieldCols - 3, 3}: config.LevelDwarf, {config.FieldCols - 2, 3}: config.LevelSack, {config.FieldCols - 1, 3}: config.LevelSack,
		}, Right},
	} {
		w, _ := newWorld(t, c.cells)
		x, y := w.Dwarf.X, w.Dwarf.Y
		if ev := run(w, c.dir, ticksPerCell); slices.Contains(ev, SackPushed) || w.Dwarf.X != x || w.Dwarf.Y != y {
			t.Errorf("%s: events %v, dwarf (%d, %d)", name, ev, w.Dwarf.X, w.Dwarf.Y)
		}
	}
}

func TestKoboldStopsAPush(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 3}: config.LevelDwarf, {1, 3}: config.LevelSack, {2, 3}: config.LevelTunnel})
	w.Tuning.KoboldSpeed = 0
	addKobold(w, 2, 3)
	run(w, Right, ticksPerCell)
	if w.Dwarf.X != 0 || w.Sacks[0].X != cell {
		t.Errorf("dwarf x %d, sack x %d", w.Dwarf.X, w.Sacks[0].X)
	}
}

func TestSlidingSackFillsBothCells(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 3}: config.LevelDwarf, {1, 3}: config.LevelSack})
	run(w, Right, 3)
	if !w.sackAt(1, 3) || !w.sackAt(2, 3) {
		t.Errorf("sack %+v should fill both cells", w.Sacks[0])
	}
}

func TestSackAndGoldStopTheShot(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 0}: config.LevelDwarf, {1, 0}: config.LevelTunnel, {2, 0}: config.LevelSack,
		{0, 3}: config.LevelTunnel, {1, 3}: config.LevelTunnel,
	})
	w.Update(None, true)
	run(w, None, 2*cell/config.ShotSpeed)
	if w.Shot != nil {
		t.Errorf("shot passed a sack: %+v", w.Shot)
	}
	if !w.stopsShot(cell*2+1, 1) {
		t.Error("a sack does not stop the shot")
	}
	w.Gold = []Gold{{X: 1, Y: 3, Ticks: config.GoldTicks}}
	if !w.stopsShot(cell+1, 3*cell+1) {
		t.Error("gold does not stop the shot")
	}
}

func TestFallingSackCrushesAKobold(t *testing.T) {
	w, s := newWorld(t, map[[2]int]byte{
		{0, 0}: config.LevelDwarf, {5, 3}: config.LevelSack,
		{5, 4}: config.LevelTunnel, {5, 5}: config.LevelTunnel,
	})
	w.Tuning.GoblinAfterTicks = -1
	w.Tuning.KoboldSpeed = 0 // it stands under the sack
	w.Kobolds = []Kobold{{X: 5 * cell, Y: 5 * cell}}
	w.Spawned = 1
	ev := run(w, None, wobbleAndFall(2))
	if !slices.Contains(ev, KoboldCrushed) {
		t.Fatalf("events %v, want KoboldCrushed", ev)
	}
	if len(w.Kobolds) != 0 || w.Killed != 1 || s.Score != config.KoboldPoints {
		t.Errorf("kobolds %v, killed %d, score %d", w.Kobolds, w.Killed, s.Score)
	}
}

func TestFallingSackCatchesTheDwarf(t *testing.T) {
	w, s := newWorld(t, map[[2]int]byte{
		{5, 3}: config.LevelSack, {5, 4}: config.LevelDwarf, {5, 5}: config.LevelTunnel,
	})
	if ev := run(w, None, wobbleAndFall(1)); !slices.Contains(ev, DwarfCaught) {
		t.Fatalf("events %v, want DwarfCaught", ev)
	}
	if !w.Caught || s.Lives != config.StartLives-1 {
		t.Errorf("caught %v, lives %d", w.Caught, s.Lives)
	}
}

func TestGoblinCutsThroughEarthThenWalksAgain(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	w.Tuning.GoblinAfterTicks = -1
	w.Tuning.KoboldSpawnTicks = 1000 * config.TPS
	addKobold(w, 6, 0)
	w.Kobolds[0].Goblin = cell / w.Tuning.KoboldSpeed // one cell's worth
	run(w, None, cell/w.Tuning.KoboldSpeed)
	if g := w.Kobolds[0]; g.X != 5*cell || g.Goblin != 0 {
		t.Fatalf("goblin %+v", g)
	}
	if w.Earth(5, 0) {
		t.Error("the goblin left earth behind")
	}
	// A walker again, it has only the cell it dug and the one it came from.
	for range 8 * cell {
		w.Update(None, false)
		if g := w.Kobolds[0]; g.X < 5*cell || g.X > 6*cell || g.Y != 0 {
			t.Fatalf("walked through earth: %+v", g)
		}
	}
}

func TestCrossingKoboldsTurnIntoAGoblinInTime(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	w.Tuning.KoboldSpawnTicks, w.Tuning.KoboldSpeed = 1000*config.TPS, 0
	addKobold(w, 6, 0)
	addKobold(w, 6, 0)
	run(w, None, w.Tuning.GoblinAfterTicks)
	if w.Kobolds[0].Goblin > 0 {
		t.Fatal("turned goblin too soon")
	}
	run(w, None, 1)
	if w.Kobolds[0].Goblin != w.Tuning.GoblinTicks {
		t.Errorf("goblin %d, want %d", w.Kobolds[0].Goblin, w.Tuning.GoblinTicks)
	}
}

func TestALoneKoboldStaysAKobold(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	w.Tuning.KoboldSpawnTicks, w.Tuning.KoboldSpeed = 1000*config.TPS, 0
	addKobold(w, 6, 0)
	run(w, None, 2*w.Tuning.GoblinAfterTicks)
	if w.Kobolds[0].Goblin > 0 {
		t.Error("turned goblin with nothing to provoke it")
	}
}

func TestAGoblinShrinksBackAndMustGrowImpatientAgain(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	w.Tuning.KoboldSpawnTicks, w.Tuning.KoboldSpeed = 1000*config.TPS, 0
	addKobold(w, 6, 0)
	w.Kobolds[0].Goblin, w.Kobolds[0].impatience = 1, 5
	run(w, None, 1)
	if g := w.Kobolds[0]; g.Goblin != 0 || g.impatience != 0 {
		t.Errorf("after its time %+v", g)
	}
}

func TestAKoboldStuckOnASackGrowsImpatient(t *testing.T) {
	cells := map[[2]int]byte{{0, 0}: config.LevelDwarf, {1, 0}: config.LevelSack, {2, 0}: config.LevelTunnel}
	w, _ := newWorld(t, cells)
	w.Tuning.KoboldSpawnTicks = 1000 * config.TPS
	w.Tuning.GoblinAfterTicks = 10
	w.earth[1] = false // the sack lies in the open, as one pushed there would
	addKobold(w, 2, 0)
	run(w, None, 40)
	if w.Kobolds[0].Goblin == 0 {
		t.Errorf("stayed a kobold: %+v", w.Kobolds[0])
	}
}

// The tests below state the original's rules, as recorded in DESIGN §6.

func TestSackFallingOntoASackInAnOpenCellDestroysIt(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 0}: config.LevelDwarf, {5, 2}: config.LevelSack, {5, 3}: config.LevelTunnel,
		{5, 4}: config.LevelSack,
	})
	w.earth[4*config.FieldCols+5] = false // the lower sack is in an open cell, on earth
	ev := run(w, None, wobbleAndFall(2))
	if !slices.Contains(ev, SackBurst) {
		t.Fatalf("events %v, want SackBurst", ev)
	}
	if len(w.Sacks) != 0 || len(w.Gold) != 1 || w.Gold[0].Y != 4 {
		t.Errorf("sacks %v, gold %v", w.Sacks, w.Gold)
	}
}

func TestSackRestsOnEarthUnderAnotherSack(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 0}: config.LevelDwarf, {5, 3}: config.LevelSack, {5, 4}: config.LevelSack,
	})
	run(w, None, wobbleAndFall(1))
	if a, b := w.Sacks[0], w.Sacks[1]; a.State != SackResting || b.State != SackResting || a.Y != 3*cell {
		t.Errorf("sacks %+v and %+v", a, b)
	}
}

func TestSackPushedSidewaysLeavesEarthAndJewelsAlone(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 0}: config.LevelDwarf, {1, 0}: config.LevelSack, {2, 0}: config.LevelJewel,
	})
	jewels := w.JewelsLeft
	run(w, Right, 2*ticksPerCell)
	if s := w.Sacks[0]; s.X != 2*cell || s.State != SackResting {
		t.Fatalf("sack %+v", s)
	}
	if !w.Earth(2, 0) || !w.Jewel(2, 0) || w.JewelsLeft != jewels {
		t.Errorf("earth %v, jewel %v, jewels left %d", w.Earth(2, 0), w.Jewel(2, 0), w.JewelsLeft)
	}
	if w.Dwarf.X != cell || w.Earth(1, 0) {
		t.Errorf("dwarf x %d, its cell earth %v", w.Dwarf.X, w.Earth(1, 0))
	}
}

func TestPushingIsAtHalfWalkingSpeed(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf, {1, 0}: config.LevelSack})
	run(w, Right, ticksPerCell)
	if w.Dwarf.X != cell/2 || w.Sacks[0].X != cell+cell/2 {
		t.Errorf("after a walking cell's time: dwarf x %d, sack x %d", w.Dwarf.X, w.Sacks[0].X)
	}
}

func TestReleasedSackFinishesItsCell(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf, {1, 0}: config.LevelSack})
	run(w, Right, 4)
	run(w, None, cell/config.SackSlideSpeed)
	if s := w.Sacks[0]; s.X != 2*cell || s.State != SackResting {
		t.Errorf("sack %+v", s)
	}
}

func TestSackPushesASackAhead(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 3}: config.LevelDwarf, {1, 3}: config.LevelSack, {2, 3}: config.LevelSack,
	})
	run(w, Right, 2*ticksPerCell)
	if a, b := w.Sacks[0], w.Sacks[1]; a.X != 2*cell || b.X != 3*cell {
		t.Errorf("sacks at %d and %d", a.X, b.X)
	}
}

func TestPushedSackFallsAtOnceOverAGap(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 3}: config.LevelDwarf, {1, 3}: config.LevelSack, {2, 4}: config.LevelTunnel,
	})
	ev := run(w, Right, 2*ticksPerCell)
	ev = append(ev, run(w, None, cell/config.SackFallSpeed+1)...)
	if slices.Contains(ev, SackWobbling) {
		t.Errorf("wobbled over the gap: %v", ev)
	}
	if s := w.Sacks[0]; s.Y != 4*cell {
		t.Errorf("sack %+v", s)
	}
}

func TestWobblingSackCanBePushedClear(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 3}: config.LevelDwarf, {1, 3}: config.LevelSack, {1, 4}: config.LevelTunnel,
	})
	run(w, None, 2)
	if w.Sacks[0].State != SackWobble {
		t.Fatalf("sack %+v", w.Sacks[0])
	}
	run(w, Right, 2*ticksPerCell)
	if s := w.Sacks[0]; s.X != 2*cell || s.Y != 3*cell || s.State != SackResting {
		t.Errorf("sack %+v", s)
	}
}

func TestSackStoppedByAKoboldTurnsBack(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf, {5, 3}: config.LevelSack})
	w.Sacks[0].State, w.Sacks[0].Slide, w.Sacks[0].X = SackSliding, Right, 5*cell+4
	w.Tuning.KoboldSpeed, w.Tuning.GoblinAfterTicks = 0, -1
	w.Kobolds = []Kobold{{X: 6*cell + 5, Y: 3 * cell}}
	w.Spawned = 1
	run(w, None, cell)
	if s := w.Sacks[0]; s.X != 5*cell || s.State != SackResting {
		t.Errorf("sack %+v", s)
	}
}

func TestWalkingKoboldPushesASackSideways(t *testing.T) {
	cells := corridor(map[[2]int]byte{{0, 3}: config.LevelDwarf}, 3, 1, 9)
	cells[[2]int{4, 3}] = config.LevelSack
	w, _ := newWorld(t, cells)
	w.earth[3*config.FieldCols+4] = false
	w.Tuning.GoblinAfterTicks, w.Tuning.KoboldSpawnTicks = -1, 1000*config.TPS
	addKobold(w, 5, 3)
	ev := run(w, None, cell/config.SackSlideSpeed)
	if !slices.Contains(ev, SackPushed) {
		t.Fatalf("events %v, want SackPushed", ev)
	}
	if s, g := w.Sacks[0], w.Kobolds[0]; s.X != 3*cell || g.X != 4*cell {
		t.Errorf("sack x %d, kobold x %d", s.X, g.X)
	}
}

func TestWalkingKoboldCantPassASackUpOrDown(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 0}: config.LevelDwarf, {5, 3}: config.LevelSack, {5, 4}: config.LevelTunnel,
	})
	w.earth[3*config.FieldCols+5] = false
	if w.walkable(5, 4, Up, false) {
		t.Error("a walker can step up into a sack")
	}
}

func TestGoblinDestroysASack(t *testing.T) {
	w, s := newWorld(t, map[[2]int]byte{{0, 5}: config.LevelDwarf, {3, 5}: config.LevelSack})
	w.Tuning.GoblinAfterTicks, w.Tuning.KoboldSpawnTicks = -1, 1000*config.TPS
	addKobold(w, 4, 5)
	w.Kobolds[0].Goblin = 100
	run(w, None, 2)
	if len(w.Sacks) != 0 || len(w.Gold) != 0 || s.Score != 0 {
		t.Errorf("sacks %v, gold %v, score %d", w.Sacks, w.Gold, s.Score)
	}
}

func TestKoboldTakesGoldWithoutScore(t *testing.T) {
	w, s := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf, {6, 3}: config.LevelTunnel})
	w.Tuning.KoboldSpeed, w.Tuning.GoblinAfterTicks = 0, -1
	addKobold(w, 6, 3)
	w.Gold = []Gold{{X: 6, Y: 3, Ticks: config.GoldTicks}}
	run(w, None, 1)
	if len(w.Gold) != 0 || s.Score != 0 {
		t.Errorf("gold %v, score %d", w.Gold, s.Score)
	}
}

func TestDigUnderGoldAndItSoonGoes(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf, {5, 3}: config.LevelTunnel})
	w.Gold = []Gold{{X: 5, Y: 3, Ticks: config.GoldTicks}}
	run(w, None, 1)
	if w.Gold[0].Ticks != config.GoldTicks-1 {
		t.Fatalf("gold %+v with earth under it", w.Gold[0])
	}
	w.earth[4*config.FieldCols+5] = false
	run(w, None, config.GoldDugUnderTicks)
	if len(w.Gold) != 0 {
		t.Errorf("gold %+v still here", w.Gold)
	}
}

func TestDwarfMovingUpBelowHoldsASack(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{5, 4}: config.LevelDwarf, {5, 3}: config.LevelSack,
	})
	if ev := run(w, Up, config.TPS); slices.Contains(ev, SackWobbling) {
		t.Fatalf("wobbled with the dwarf pressing up under it: %v", ev)
	}
	if ev := run(w, None, 1); !slices.Contains(ev, SackWobbling) {
		t.Errorf("no wobble once let go: %v", ev)
	}
}

func TestFallingSackDestroysAJewelInAnOpenCell(t *testing.T) {
	w, s := newWorld(t, map[[2]int]byte{
		{0, 0}: config.LevelDwarf, {5, 3}: config.LevelSack, {5, 4}: config.LevelJewel,
	})
	w.earth[4*config.FieldCols+5] = false // as a digger leaves it
	jewels := w.JewelsLeft
	run(w, None, wobbleAndFall(1))
	if w.Jewel(5, 4) || w.JewelsLeft != jewels-1 || s.Score != 0 {
		t.Errorf("jewel %v, left %d of %d, score %d", w.Jewel(5, 4), w.JewelsLeft, jewels, s.Score)
	}
}

func TestDwarfAboveAFallingSackIsSafe(t *testing.T) {
	w, s := newWorld(t, map[[2]int]byte{
		{5, 3}: config.LevelDwarf, {6, 3}: config.LevelTunnel, {5, 4}: config.LevelTunnel,
	})
	w.Sacks = []Sack{{X: 5 * cell, Y: 3*cell + 4, State: SackFalling}}
	run(w, None, 2)
	if w.Caught || s.Lives != config.StartLives {
		t.Errorf("caught %v, lives %d", w.Caught, s.Lives)
	}
}

func TestKoboldClimbingTurnsBackFromAFallingSack(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	w.Tuning.GoblinAfterTicks = -1
	w.Sacks = []Sack{{X: 7 * cell, Y: 0, State: SackFalling}}
	w.Kobolds = []Kobold{{X: 7 * cell, Y: 6*cell + 2, Moving: Up}}
	w.Spawned = 1
	w.earth[1*config.FieldCols+7] = false
	w.moveSacks()
	if w.Kobolds[0].Moving != Down {
		t.Errorf("kobold %+v", w.Kobolds[0])
	}
}

func TestRespawnClearsGoldAndUnsettledSacks(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 0}: config.LevelDwarf, {5, 3}: config.LevelSack, {7, 3}: config.LevelSack,
	})
	w.Gold = []Gold{{X: 1, Y: 1, Ticks: config.GoldTicks}}
	w.Sacks[1].State = SackWobble
	w.Respawn()
	if len(w.Gold) != 0 || len(w.Sacks) != 1 || w.Sacks[0].X != 5*cell {
		t.Errorf("gold %v, sacks %v", w.Gold, w.Sacks)
	}
}

func TestBlockedPushDropsAWobblingSackAtOnce(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{
		{0, 3}: config.LevelDwarf, {1, 3}: config.LevelSack, {1, 4}: config.LevelTunnel, {2, 3}: config.LevelTunnel,
	})
	w.Tuning.KoboldSpeed, w.Tuning.GoblinAfterTicks = 0, -1
	addKobold(w, 2, 3)
	run(w, None, 2)
	w.Update(Right, false)
	if s := w.Sacks[0]; s.State != SackFalling || s.X != cell {
		t.Errorf("sack %+v", s)
	}
}
