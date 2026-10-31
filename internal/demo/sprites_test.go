package demo

import (
	"os"
	"testing"

	"github.com/comprofix/gameskeleton/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/gameskeleton/internal/spritegrid"
)

// TestSprites checks the demo's sprite source parses and packs, and has
// every sprite the play scene draws.
func TestSprites(t *testing.T) {
	f, err := os.Open("../../assets/sprites/demo.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sprites, err := spritegrid.Parse(f, resurrect64.Palette)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := spritegrid.Pack(sprites, resurrect64.Palette); err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, s := range sprites {
		have[s.Name] = true
	}
	for _, name := range []string{"player", "player_hurt", "dot", "hazard", "heart"} {
		if !have[name] {
			t.Errorf("no sprite named %q", name)
		}
	}
}
