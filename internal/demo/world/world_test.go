package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/session"
)

// quiet is a game with no hazards or dots, for adding them by hand.
func quiet(t *testing.T) *World {
	t.Helper()
	tu := ForSkill("easy")
	tu.Hazards, tu.Dots = 0, 0
	return New(session.New(config.Skills[0], 0), 9, 7, tu, rand.New(rand.NewPCG(1, 2)))
}

func run(w *World, dir Dir, ticks int) (all []Event) {
	for range ticks {
		all = append(all, w.Update(dir)...)
	}
	return all
}

func count(evs []Event, e Event) int {
	n := 0
	for _, x := range evs {
		if x == e {
			n++
		}
	}
	return n
}

func TestStartsInTheMiddle(t *testing.T) {
	w := quiet(t)
	if w.Player.Pos != (Point{4, 3}) || w.Session.Lives != 3 || w.TicksLeft != 30*config.TPS {
		t.Errorf("start: player %v, hearts %d, ticks %d", w.Player.Pos, w.Session.Lives, w.TicksLeft)
	}
}

func TestStepAndHold(t *testing.T) {
	w := quiet(t)
	evs := run(w, Right, 1)
	if w.Player.Pos != (Point{5, 3}) || count(evs, Stepped) != 1 {
		t.Fatalf("a press did not step at once: %v", w.Player.Pos)
	}
	// Held: the next step comes StepTicks later.
	n := w.Tuning.StepTicks
	run(w, Right, n-1)
	if w.Player.Pos.X != 5 {
		t.Fatalf("stepped again after %d ticks, want %d", n-1, n)
	}
	run(w, Right, 1)
	if w.Player.Pos.X != 6 {
		t.Fatalf("did not step again after %d ticks", n)
	}
	// Letting go and pressing again steps at once.
	run(w, None, 1)
	run(w, Left, 1)
	if w.Player.Pos.X != 5 || !w.FacesLeft {
		t.Errorf("after a new press: x %d, facing left %v", w.Player.Pos.X, w.FacesLeft)
	}
}

func TestEdgesStopThePlayer(t *testing.T) {
	w := quiet(t)
	evs := run(w, Up, 100)
	if w.Player.Pos != (Point{4, 0}) {
		t.Errorf("player at %v, want the top edge", w.Player.Pos)
	}
	if n := count(evs, Stepped); n != 3 {
		t.Errorf("%d steps, want 3 (no steps into the edge)", n)
	}
}

func TestPickUpADot(t *testing.T) {
	w := quiet(t)
	w.Dots = []Point{{5, 3}, {0, 0}}
	evs := run(w, Right, 1)
	if count(evs, Picked) != 1 || w.DotsTaken != 1 || w.Session.Score != w.Tuning.DotPoints {
		t.Fatalf("events %v, dots %d, score %d", evs, w.DotsTaken, w.Session.Score)
	}
	if len(w.Dots) != 2 || w.Dots[0] == w.Player.Pos || w.Dots[0] == w.Dots[1] {
		t.Errorf("the dot came back at %v (player %v, other dot %v)", w.Dots[0], w.Player.Pos, w.Dots[1])
	}
}

func TestHazardBouncesOffTheEdge(t *testing.T) {
	w := quiet(t)
	w.Hazards = []Hazard{{Mover: Mover{Pos: Point{7, 0}}, Dir: Point{1, 0}, wait: 1}}
	var xs []int
	for range 4 {
		run(w, None, w.Tuning.HazardTicks)
		xs = append(xs, w.Hazards[0].Pos.X)
	}
	if want := []int{8, 7, 6, 5}; !slices.Equal(xs, want) {
		t.Errorf("hazard x went %v, want %v", xs, want)
	}
}

func TestBumpCostsAHeartThenIsSafe(t *testing.T) {
	w := quiet(t)
	w.Hazards = []Hazard{{Mover: Mover{Pos: Point{5, 3}}, Dir: Point{0, 1}, wait: 1000}}
	evs := run(w, Right, 1)
	if count(evs, Bumped) != 1 || w.Session.Lives != 2 {
		t.Fatalf("stepping onto a hazard: events %v, hearts %d", evs, w.Session.Lives)
	}
	evs = run(w, None, w.Tuning.SafeTicks-1)
	if count(evs, Bumped) != 0 {
		t.Error("bumped again while safe")
	}
	evs = run(w, None, 1)
	if count(evs, Bumped) != 1 || w.Session.Lives != 1 {
		t.Errorf("not bumped once safety ran out: hearts %d", w.Session.Lives)
	}
}

func TestPassingThroughAHazardBumps(t *testing.T) {
	w := quiet(t)
	// The hazard moves left into the player's cell as the player steps
	// right into the hazard's: they swap.
	w.Hazards = []Hazard{{Mover: Mover{Pos: Point{5, 3}}, Dir: Point{-1, 0}, wait: 1}}
	evs := run(w, Right, 1)
	if count(evs, Bumped) != 1 {
		t.Errorf("swapping cells with a hazard did not bump: player %v, hazard %v", w.Player.Pos, w.Hazards[0].Pos)
	}
}

func TestOutOfHeartsEndsTheGame(t *testing.T) {
	w := quiet(t)
	w.Session.Lives = 1
	w.Hazards = []Hazard{{Mover: Mover{Pos: Point{5, 3}}, wait: 1000}}
	evs := run(w, Right, 1)
	if !w.Over || count(evs, OutOfHearts) != 1 {
		t.Fatalf("over %v, events %v", w.Over, evs)
	}
	if evs := run(w, Left, 10); len(evs) != 0 || w.Player.Pos.X != 5 {
		t.Error("the game carried on after it was over")
	}
	if w.HeartBonus() != 0 {
		t.Errorf("heart bonus %d with no hearts", w.HeartBonus())
	}
}

func TestTimeUp(t *testing.T) {
	w := quiet(t)
	evs := run(w, None, w.Tuning.Ticks)
	if !w.Over || w.TicksLeft != 0 || count(evs, TimeUp) != 1 {
		t.Errorf("after %d ticks: over %v, left %d", w.Tuning.Ticks, w.Over, w.TicksLeft)
	}
	if w.HeartBonus() != 3*w.Tuning.HeartPoints {
		t.Errorf("heart bonus %d", w.HeartBonus())
	}
}

func TestNewGameIsFairAndRepeatable(t *testing.T) {
	newGame := func(seed uint64) *World {
		return New(session.New(config.Skills[1], 0), 19, 10, ForSkill("hard"), rand.New(rand.NewPCG(seed, 7)))
	}
	w := newGame(42)
	if len(w.Hazards) != 5 || len(w.Dots) != 3 {
		t.Fatalf("%d hazards, %d dots", len(w.Hazards), len(w.Dots))
	}
	for _, h := range w.Hazards {
		if dist(h.Pos, w.Player.Pos) < w.Tuning.SafeRadius {
			t.Errorf("hazard at %v is too close to the player at %v", h.Pos, w.Player.Pos)
		}
		if abs(h.Dir.X)+abs(h.Dir.Y) != 1 {
			t.Errorf("hazard direction %v", h.Dir)
		}
	}
	other := newGame(42)
	if !slices.Equal(w.Dots, other.Dots) || w.Hazards[0] != other.Hazards[0] {
		t.Error("the same seed gave a different board")
	}
}

func TestSkills(t *testing.T) {
	easy, hard := ForSkill("easy"), ForSkill("hard")
	if !(hard.Ticks < easy.Ticks && hard.Hazards > easy.Hazards && hard.HazardTicks < easy.HazardTicks) {
		t.Errorf("hard is not harder: easy %+v, hard %+v", easy, hard)
	}
	if ForSkill("nonsense") != easy {
		t.Error("an unknown skill is not Easy")
	}
	for _, sk := range config.Skills {
		if ForSkill(sk.ID).Ticks == 0 {
			t.Errorf("skill %q has no tuning", sk.ID)
		}
	}
}
