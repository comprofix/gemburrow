package config

// Skill is a difficulty level the player picks before a game. Its fields
// adjust the level ramp (see TuningFor). Normal changes nothing.
type Skill struct {
	ID   string // stable key, used in saved files
	Name string // shown in menus
	Desc string // one line shown under the menu

	ExtraKobolds    int // added to the kobolds in a level
	ExtraAtOnce     int // added to the most on the field together
	ExtraRarity     int // added to how rarely a walking kobold steps extra
	SpawnPercent    int // of the time between kobolds entering
	RechargePercent int // of the shot's recharge
	ExtraPatience   int // added to the crossings before a kobold turns goblin
}

// Skills are the skill levels, easiest first (DESIGN §5).
var Skills = []Skill{
	{
		ID: "easy", Name: "Easy", Desc: "Fewer, slower kobolds and a quicker shot.",
		ExtraKobolds: -1, ExtraAtOnce: -1, ExtraRarity: 5,
		SpawnPercent: 125, RechargePercent: 75, ExtraPatience: 2,
	},
	{
		ID: "normal", Name: "Normal", Desc: "The classic climb in danger.",
		SpawnPercent: 100, RechargePercent: 100,
	},
	{
		ID: "hard", Name: "Hard", Desc: "More, faster kobolds and a slower shot.",
		ExtraKobolds: 2, ExtraAtOnce: 1, ExtraRarity: -3,
		SpawnPercent: 75, RechargePercent: 125, ExtraPatience: -1,
	},
}
