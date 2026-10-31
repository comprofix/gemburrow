package config

// Level layout characters.
const (
	LevelEarth  = '#' // solid earth
	LevelTunnel = '=' // an open tunnel
	LevelJewel  = '*' // a jewel set in earth
	LevelDwarf  = 'D' // the dwarf's start, in an open tunnel
)

// Levels are the original layouts, FieldRows rows of FieldCols characters.
var Levels = [][]string{
	{
		"##*#####==#####*##",
		"#####*##==##*#####",
		"##*#####==#####*##",
		"#####*##==##*#####",
		"========D=========",
		"##*#####==#####*##",
		"#####*##==##*#####",
		"##*#####==#####*##",
		"#####*##==##*#####",
		"##*#####==#####*##",
	},
}

// LevelFor is the layout of level n, counting from 1. After the last, the
// layouts repeat (DESIGN D9).
func LevelFor(n int) []string {
	return Levels[(max(n, 1)-1)%len(Levels)]
}
