package config

// Skill is a difficulty level the player picks before a game. Its fields
// adjust the level ramp (see TuningFor). Normal changes nothing.
type Skill struct {
	ID   string // stable key, used in saved files
	Name string // shown in menus
	Desc string // one line shown under the menu

	ExtraGoblins    int // added to the goblins in a level
	ExtraAtOnce     int // added to the most on the field together
	ExtraRarity     int // added to how rarely a walking goblin steps extra
	SpawnPercent    int // of the time between goblins entering
	RechargePercent int // of the shot's recharge
	DiggerPercent   int // of the time between a goblin turning digger
}

// Skills are the skill levels, easiest first (DESIGN §5).
var Skills = []Skill{
	{
		ID: "easy", Name: "Easy", Desc: "Fewer, slower goblins and a quicker shot.",
		ExtraGoblins: -1, ExtraAtOnce: -1, ExtraRarity: 5,
		SpawnPercent: 125, RechargePercent: 75, DiggerPercent: 125,
	},
	{
		ID: "normal", Name: "Normal", Desc: "The classic climb in danger.",
		SpawnPercent: 100, RechargePercent: 100, DiggerPercent: 100,
	},
	{
		ID: "hard", Name: "Hard", Desc: "More, faster goblins and a slower shot.",
		ExtraGoblins: 2, ExtraAtOnce: 1, ExtraRarity: -3,
		SpawnPercent: 75, RechargePercent: 125, DiggerPercent: 75,
	},
}
