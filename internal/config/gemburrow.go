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
)
