package config

// The field, in pixels on the pixel look's screen, whose coordinates the
// game uses in both looks (DESIGN §3, D2).
const (
	CellSize  = 32
	FieldCols = 15
	FieldRows = 10
	HUDHeight = 40
	FieldX    = 80
	FieldY    = HUDHeight
	// HUDX is the HUD's margin. It runs wider than the field, which is too
	// narrow to hold all of it.
	HUDX = 32
)

// Tuning values (DESIGN §5, D4).
const (
	// DwarfSpeed is in pixels per tick, and must divide CellSize.
	DwarfSpeed  = 2
	JewelPoints = 25
	StartLives  = 3
	// LevelClearTicks is how long the level clear banner stays up.
	LevelClearTicks = 2 * TPS

	GoblinPoints   = 250
	ExtraLifeEvery = 20000
	// ShotSpeed is in pixels per tick.
	ShotSpeed = 4
	// ShotSize is the shot's sprite, which is square.
	ShotSize = 8
	// ContactInset trims each side of a 32-pixel box when the dwarf and a
	// goblin touch, so that only a real overlap counts.
	ContactInset = 6
	// SpawnClearCells holds a goblin back while the dwarf is this near the
	// entrance, counted in cells along the grid.
	SpawnClearCells = 2
	// CaughtTicks is the pause after the dwarf is caught, and GameOverTicks
	// how long the game over banner stays up.
	CaughtTicks   = 3 * TPS / 2
	GameOverTicks = 3 * TPS

	// Sacks and gold (DESIGN §6, D6). The speeds are in pixels per tick, and
	// must divide CellSize. A sack bursts if it falls more than one cell.
	// SackSlideSpeed is half the dwarf's walking speed, and SackFallSpeed
	// twice it. GoldDugUnderTicks is what gold is cut to once the cell under
	// it is dug out.
	SackWobbleTicks   = 1 * TPS
	SackFallSpeed     = 4
	SackSlideSpeed    = 1
	GoldTicks         = 5 * TPS
	GoldDugUnderTicks = 4 * TPS / 5
	GoldPoints        = 500
)

// Tuning is what difficulty and the level ramp change (DESIGN §5).
type Tuning struct {
	// GoblinSpeed is in pixels per tick, and must divide CellSize.
	GoblinSpeed int
	// GoblinExtraOneIn gives a walking goblin an extra pixel's move, one tick
	// in this many; 0 means never.
	GoblinExtraOneIn  int
	GoblinTotal       int // goblins in a level
	GoblinsAtOnce     int // the most on the field together
	GoblinSpawnTicks  int // between goblins entering
	ShotRechargeTicks int // from firing until the next shot
	DiggerEveryTicks  int // between a goblin turning digger; 0 means never
	DiggerTicks       int // how long it stays one (DESIGN D8)
}

// The level ramp (DESIGN §5, §11). It stops rising at RampLevels. Counts the
// original gives in its game ticks are converted by originalTicks.
const (
	RampLevels = 10

	rampGoblinSpeed   = 1
	rampMinGoblins    = 1
	rampMinAtOnce     = 2
	rampMinExtraOneIn = 2
	// A goblin turns digger this often on level 1, less rampDiggerFall for
	// each level after it.
	rampDiggerEvery = 8 * TPS
	rampDiggerFall  = 4 * TPS / 9
)

// originalTicks converts a count of the original's game ticks, of 80 ms at
// its reference speed, to this game's ticks.
func originalTicks(n int) int { return n * TPS * 2 / 25 }

// TuningFor is the tuning for a level, counting from 1, at a skill.
func TuningFor(s Skill, level int) Tuning {
	l := min(max(level, 1), RampLevels)
	atOnce := 4
	switch {
	case l == 1:
		atOnce = 3
	case l >= 8:
		atOnce = 5
	}
	return Tuning{
		GoblinSpeed:       rampGoblinSpeed,
		GoblinExtraOneIn:  max(15-l+s.ExtraRarity, rampMinExtraOneIn),
		GoblinTotal:       max(l+5+s.ExtraGoblins, rampMinGoblins),
		GoblinsAtOnce:     max(atOnce+s.ExtraAtOnce, rampMinAtOnce),
		GoblinSpawnTicks:  originalTicks(45-2*l) * s.SpawnPercent / 100,
		ShotRechargeTicks: originalTicks(60+3*l) * s.RechargePercent / 100,
		DiggerEveryTicks:  (rampDiggerEvery - (l-1)*rampDiggerFall) * s.DiggerPercent / 100,
		DiggerTicks:       originalTicks(30 + 2*l),
	}
}

// BaseTuning is level 1 at Normal, for tests.
var BaseTuning = TuningFor(Skills[1], 1)
