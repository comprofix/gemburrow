package config

// Skill is a difficulty level the player picks before a game.
// Games replace these with their own skills and tuning.
type Skill struct {
	ID   string // stable key, used in saved files
	Name string // shown in menus
	Desc string // one line shown under the menu
}

// Skills are the skill levels, easiest first.
var Skills = []Skill{
	{ID: "easy", Name: "Easy", Desc: "A gentle start: slower and more time."},
	{ID: "hard", Name: "Hard", Desc: "Faster, with less time on the clock."},
}
