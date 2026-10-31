package world

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/comprofix/gemburrow/internal/config"
	"github.com/comprofix/gemburrow/internal/session"
)

const (
	cell         = config.CellSize
	ticksPerCell = cell / config.DwarfSpeed
)

// layout is an all-earth field with the given cells set, and a far jewel so
// the level doesn't clear unless a test places its own.
func layout(cells map[[2]int]byte) []string {
	rows := make([][]byte, config.FieldRows)
	for y := range rows {
		rows[y] = []byte(strings.Repeat(string(rune(config.LevelEarth)), config.FieldCols))
	}
	rows[config.FieldRows-1][config.FieldCols-1] = config.LevelJewel
	rows[config.FieldRows-2][config.FieldCols-1] = config.LevelSpawn
	for p, c := range cells {
		rows[p[1]][p[0]] = c
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = string(r)
	}
	return out
}

func newWorld(t *testing.T, cells map[[2]int]byte) (*World, *session.Session) {
	t.Helper()
	l, err := Parse(layout(cells))
	if err != nil {
		t.Fatal(err)
	}
	s := session.New(config.Skills[0], config.StartLives)
	tuning := config.BaseTuning
	tuning.GoblinExtraOneIn = 0 // an exact pace, so that tests can count ticks
	return New(s, l, tuning, rand.New(rand.NewPCG(1, 2))), s
}

func run(w *World, dir Dir, ticks int) []Event {
	var all []Event
	for range ticks {
		all = append(all, w.Update(dir, false)...)
	}
	return all
}

func TestThereAreEightLayouts(t *testing.T) {
	if len(config.Levels) != 8 {
		t.Errorf("%d layouts, want 8 (DESIGN D9)", len(config.Levels))
	}
}

// TestGoblinsCanReachTheDwarfInEveryLevel walks the open tunnels from the
// entrance, as a goblin that isn't digging would.
func TestGoblinsCanReachTheDwarfInEveryLevel(t *testing.T) {
	for i, rows := range config.Levels {
		l, err := Parse(rows)
		if err != nil {
			t.Fatal(err)
		}
		s := session.New(config.Skills[0], config.StartLives)
		w := New(s, l, config.BaseTuning, rand.New(rand.NewPCG(1, 2)))
		w.measure(false)
		if d := w.dist[l.spawnY*config.FieldCols+l.spawnX]; d < 0 {
			t.Errorf("level %d: no tunnel from the entrance to the dwarf:\n%v", i+1, w)
		}
	}
}

func TestEveryLevelParses(t *testing.T) {
	for i, rows := range config.Levels {
		if _, err := Parse(rows); err != nil {
			t.Errorf("level %d: %v", i+1, err)
		}
	}
}

func TestParseRejects(t *testing.T) {
	good := layout(map[[2]int]byte{{0, 0}: config.LevelDwarf})
	mutate := func(f func(r []string)) []string {
		r := slices.Clone(good)
		f(r)
		return r
	}
	for name, rows := range map[string][]string{
		"too few rows": good[:config.FieldRows-1],
		"short row":    mutate(func(r []string) { r[1] = "##" }),
		"bad cell":     mutate(func(r []string) { r[1] = "?" + r[1][1:] }),
		"no dwarf":     mutate(func(r []string) { r[0] = "#" + r[0][1:] }),
		"two dwarves":  mutate(func(r []string) { r[1] = "D" + r[1][1:] }),
		"no spawn":     mutate(func(r []string) { r[config.FieldRows-2] = r[0][1:] + "#" }),
		"two spawns":   mutate(func(r []string) { r[1] = "G" + r[1][1:] }),
		"no jewels": mutate(func(r []string) {
			last := len(r) - 1
			r[last] = r[last][:config.FieldCols-1] + "#"
		}),
	} {
		if _, err := Parse(rows); err == nil {
			t.Errorf("%s: parsed", name)
		}
	}
}

func TestDiggingOpensEarthOnceInside(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	run(w, Right, ticksPerCell/2-1)
	if !w.Earth(1, 0) {
		t.Error("dug before the dwarf's centre was inside")
	}
	if ev := run(w, Right, 1); !slices.Contains(ev, Dug) {
		t.Errorf("events %v, want Dug", ev)
	}
	if w.Earth(1, 0) {
		t.Error("not dug with the dwarf's centre inside")
	}
	if ev := run(w, Right, ticksPerCell/2); slices.Contains(ev, Dug) {
		t.Errorf("dug twice: %v", ev)
	}
	if w.Dwarf.X != cell || w.Dwarf.Y != 0 {
		t.Errorf("dwarf at (%d, %d)", w.Dwarf.X, w.Dwarf.Y)
	}
}

func TestTurnsOnlyAtCellCentres(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	run(w, Right, 3)
	run(w, Down, ticksPerCell-3)
	if w.Dwarf.X != cell || w.Dwarf.Y != 0 {
		t.Fatalf("turned mid-cell: (%d, %d)", w.Dwarf.X, w.Dwarf.Y)
	}
	run(w, Down, 1)
	if w.Dwarf.X != cell || w.Dwarf.Y != config.DwarfSpeed {
		t.Errorf("did not turn at the centre: (%d, %d)", w.Dwarf.X, w.Dwarf.Y)
	}
}

func TestFinishesTheCellWithNothingPushed(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	run(w, Right, 1)
	run(w, None, ticksPerCell+5)
	if w.Dwarf.X != cell {
		t.Errorf("stopped at x %d", w.Dwarf.X)
	}
}

func TestReversesMidCell(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf})
	run(w, Right, 3)
	run(w, Left, 3)
	if w.Dwarf.X != 0 {
		t.Errorf("x %d after reversing", w.Dwarf.X)
	}
	if !w.Earth(1, 0) {
		t.Error("dug a cell the dwarf never reached")
	}
	if !w.Dwarf.FacesLeft {
		t.Error("not facing left")
	}
}

func TestFieldEdgesStopTheDwarf(t *testing.T) {
	for _, c := range []struct {
		name  string
		start [2]int
		dir   Dir
	}{
		{"top", [2]int{0, 0}, Up},
		{"left", [2]int{0, 0}, Left},
		{"right", [2]int{config.FieldCols - 1, 0}, Right},
		{"bottom", [2]int{0, config.FieldRows - 1}, Down},
	} {
		w, _ := newWorld(t, map[[2]int]byte{c.start: config.LevelDwarf})
		x, y := w.Dwarf.X, w.Dwarf.Y
		run(w, c.dir, ticksPerCell)
		if w.Dwarf.X != x || w.Dwarf.Y != y {
			t.Errorf("%s: moved to (%d, %d)", c.name, w.Dwarf.X, w.Dwarf.Y)
		}
		if w.Dwarf.Facing != c.dir {
			t.Errorf("%s: facing %v", c.name, w.Dwarf.Facing)
		}
	}
}

func TestJewelScoresAndClearsTheLevel(t *testing.T) {
	w, s := newWorld(t, map[[2]int]byte{
		{0, 0}: config.LevelDwarf,
		{1, 0}: config.LevelJewel,
		{2, 0}: config.LevelJewel,
		{config.FieldCols - 1, config.FieldRows - 1}: config.LevelEarth,
	})
	if w.JewelsLeft != 2 {
		t.Fatalf("jewels %d", w.JewelsLeft)
	}
	ev := run(w, Right, ticksPerCell)
	if !slices.Contains(ev, JewelTaken) || slices.Contains(ev, LevelCleared) {
		t.Errorf("first jewel: %v", ev)
	}
	if s.Score != config.JewelPoints {
		t.Errorf("score %d", s.Score)
	}
	ev = run(w, Right, ticksPerCell)
	if !slices.Contains(ev, LevelCleared) || !w.Cleared() {
		t.Errorf("last jewel: %v", ev)
	}
	if s.Score != 2*config.JewelPoints || w.JewelsTaken != 2 {
		t.Errorf("score %d, taken %d", s.Score, w.JewelsTaken)
	}
	x := w.Dwarf.X
	if ev := run(w, Right, 5); len(ev) != 0 || w.Dwarf.X != x {
		t.Errorf("still running after the clear: %v, x %d", ev, w.Dwarf.X)
	}
}

func TestSameInputSameWorld(t *testing.T) {
	play := func() string {
		w, _ := newWorld(t, map[[2]int]byte{{4, 4}: config.LevelDwarf})
		for _, d := range []Dir{Up, Right, Right, Down, Left, None, Down} {
			run(w, d, ticksPerCell+3)
		}
		return w.String()
	}
	if a, b := play(), play(); a != b {
		t.Errorf("runs differ:\n%s\n%s", a, b)
	}
}

func TestTheDwarfDigsOnlyWhileCuttingIntoEarth(t *testing.T) {
	w, _ := newWorld(t, map[[2]int]byte{{0, 0}: config.LevelDwarf, {1, 0}: config.LevelTunnel})
	w.Update(Right, false)
	if w.Dwarf.Digging {
		t.Error("digging along a tunnel")
	}
	run(w, Right, ticksPerCell-1)
	w.Update(Right, false)
	if !w.Dwarf.Digging {
		t.Error("not digging into earth")
	}
	run(w, Right, ticksPerCell-1)
	w.Update(None, false)
	if w.Dwarf.Digging {
		t.Error("still digging once stopped")
	}
}
