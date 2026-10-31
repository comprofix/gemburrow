package game

import "github.com/comprofix/gameskeleton/internal/demo"

// The demo's place in the game. tools/newgame deletes this file along with
// internal/demo; a game wires in its own play scene, sounds and sprite
// sheets in a file like this one.
func init() {
	content = Content{
		NewPlay: demo.NewPlay,
		Sounds:  demo.Sounds,
		Sheets:  []string{"demo"},
	}
}
