package play

import (
	"slices"
	"testing"

	"github.com/comprofix/gemburrow/internal/config"
)

// isFlame reports whether any mote has a flame's colour.
func isFlame(motes []mote) bool {
	for _, m := range motes {
		if m.c != smokeColour(0) && m.c != smokeColour(0.5) && m.c != smokeColour(1) {
			return true
		}
	}
	return false
}

func TestPuffEndsOnTime(t *testing.T) {
	if len(puff(0, 0, 0)) == 0 || len(puff(0, 0, config.PuffTicks-1)) == 0 {
		t.Error("the puff is missing during its life")
	}
	if puff(0, 0, config.PuffTicks) != nil {
		t.Error("the puff outlives its time")
	}
}

func TestBreathIsTheSameEachTime(t *testing.T) {
	for ticks := range int64(3 * config.BreathCycleTicks) {
		if !slices.Equal(breath(32, 64, true, ticks, 2, true), breath(32, 64, true, ticks, 2, true)) {
			t.Fatalf("tick %d differs", ticks)
		}
	}
}

func TestFlameComesOnlyEveryFewCyclesAndWithRoom(t *testing.T) {
	during := int64(config.NoseSmokeTicks + config.FlameTicks/2)
	if !isFlame(breath(0, 0, false, during, 0, true)) {
		t.Error("no flame in a fiery cycle")
	}
	if isFlame(breath(0, 0, false, during+config.BreathCycleTicks, 0, true)) {
		t.Error("flame in a plain cycle")
	}
	if len(breath(0, 0, false, during, 0, false)) != 0 {
		t.Error("something drawn where the flame has no room")
	}
	if len(breath(0, 0, false, config.NoseSmokeTicks/2, 0, false)) == 0 {
		t.Error("no smoke from the nose")
	}
}

func TestFlameFacesTheKobold(t *testing.T) {
	during := int64(config.NoseSmokeTicks + config.FlameTicks/2)
	for _, m := range breath(0, 0, false, during, 0, true) {
		if m.x < config.CellSize/2 {
			t.Fatalf("right-facing flame at %v", m.x)
		}
	}
	for _, m := range breath(0, 0, true, during, 0, true) {
		if m.x > config.CellSize/2 {
			t.Fatalf("left-facing flame at %v", m.x)
		}
	}
}

func TestFlameStaysWithinTheNextCell(t *testing.T) {
	for ticks := range int64(config.BreathCycleTicks) {
		for _, m := range breath(0, 0, false, ticks, 0, true) {
			if m.x-m.size/2 > 2*config.CellSize {
				t.Fatalf("tick %d: mote reaches %v", ticks, m.x)
			}
		}
	}
}
