package world

import "github.com/comprofix/gameskeleton/internal/config"

// Tuning is one skill's settings for Dot Dash. The demo keeps
// its tuning here rather than in internal/config so that removing the demo
// removes it too.
type Tuning struct {
	Ticks       int // the game's length
	Hearts      int // bumps allowed
	Hazards     int
	HazardTicks int // ticks a hazard takes to move one cell
	Dots        int // dots on the board at once
	DotPoints   int
	HeartPoints int // bonus for each heart left at the end
	StepTicks   int // ticks between steps while a direction is held
	SafeTicks   int // after a bump, ticks before the player can be bumped again
	SafeRadius  int // hazards start at least this many cells from the player
}

// base is what both skills share.
var base = Tuning{
	Hearts:      3,
	Dots:        3,
	DotPoints:   10,
	HeartPoints: 50,
	StepTicks:   config.TPS / 8,
	SafeTicks:   config.TPS,
	SafeRadius:  4,
}

// ForSkill is the tuning for a skill, by its ID. An unknown ID gets Easy.
func ForSkill(id string) Tuning {
	t := base
	switch id {
	case "hard":
		t.Ticks, t.Hazards, t.HazardTicks = 20*config.TPS, 5, config.TPS/6
	default:
		t.Ticks, t.Hazards, t.HazardTicks = 30*config.TPS, 3, config.TPS/3
	}
	return t
}
