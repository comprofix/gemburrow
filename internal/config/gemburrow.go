package config

// The field, in pixels on the pixel look's screen, whose coordinates the
// game uses in both looks (DESIGN §3, D2).
const (
	CellSize  = 32
	FieldCols = 18
	FieldRows = 10
	HUDHeight = 40
	FieldX    = 32
	FieldY    = HUDHeight
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
	GoblinSpeed       int
	GoblinTotal       int // goblins in a level
	GoblinsAtOnce     int // the most on the field together
	GoblinSpawnTicks  int // between goblins entering
	ShotRechargeTicks int // from firing until the next shot
	DiggerEveryTicks  int // between a goblin turning digger; 0 means never
	DiggerTicks       int // how long it stays one (DESIGN D8)
}

// BaseTuning is the only tuning until the skill tables are filled in.
var BaseTuning = Tuning{
	GoblinSpeed:       1,
	GoblinTotal:       6,
	GoblinsAtOnce:     3,
	GoblinSpawnTicks:  3 * TPS,
	ShotRechargeTicks: 2 * TPS,
	DiggerEveryTicks:  8 * TPS,
	DiggerTicks:       4 * TPS,
}
