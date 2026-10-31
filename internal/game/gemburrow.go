package game

import "github.com/comprofix/gemburrow/internal/play"

// Gem Burrow's place in the game.
func init() {
	content = Content{
		NewPlay: play.New,
		Sounds:  play.Sounds,
		Sheets:  []string{"gemburrow"},
	}
}
