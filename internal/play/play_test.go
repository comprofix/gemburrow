package play

import (
	"fmt"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gemburrow/internal/config"
	"github.com/comprofix/gemburrow/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/gemburrow/internal/scene"
	"github.com/comprofix/gemburrow/internal/scene/scenetest"
	"github.com/comprofix/gemburrow/internal/spritegrid"
)

const ticksPerCell = config.CellSize / config.DwarfSpeed

func start(t *testing.T) (*scenetest.Harness, *Play) {
	h := scenetest.New(t, 42)
	h.Ctx.NewPlay = New
	p := New(h.Ctx, config.Skills[0]).(*Play)
	h.Start(p)
	h.Ticks(scene.FadeTicks)
	return h, p
}

// walk moves the dwarf n cells, stopping if the level clears.
func walk(h *scenetest.Harness, p *Play, key ebiten.Key, n int) {
	for range n {
		if p.World().Cleared() {
			return
		}
		h.Run(scenetest.Hold(ticksPerCell, key))
	}
}

// sweep walks over every cell of the field from the first level's start.
func sweep(h *scenetest.Harness, p *Play) {
	walk(h, p, ebiten.KeyArrowUp, p.World().Dwarf.Y/config.CellSize)
	walk(h, p, ebiten.KeyArrowLeft, p.World().Dwarf.X/config.CellSize)
	for y := range config.FieldRows {
		key := ebiten.KeyArrowRight
		if y%2 == 1 {
			key = ebiten.KeyArrowLeft
		}
		walk(h, p, key, config.FieldCols-1)
		walk(h, p, ebiten.KeyArrowDown, 1)
	}
}

func TestClearingALevelGoesToTheNext(t *testing.T) {
	h, p := start(t)
	jewels := p.World().JewelsLeft
	sweep(h, p)
	if !p.World().Cleared() {
		t.Fatalf("not cleared:\n%v", p.World())
	}
	score := jewels * config.JewelPoints
	if h.Ctx.Session.Score != score {
		t.Errorf("score %d, want %d", h.Ctx.Session.Score, score)
	}
	h.Ticks(config.LevelClearTicks)
	if p.Level() != 2 || p.World().Cleared() {
		t.Errorf("level %d, cleared %v", p.Level(), p.World().Cleared())
	}
	if h.Ctx.Session.Score != score {
		t.Errorf("score %d after the next level started, want %d", h.Ctx.Session.Score, score)
	}
}

func TestWASDMoves(t *testing.T) {
	h, p := start(t)
	x := p.World().Dwarf.X
	h.Run(scenetest.Hold(ticksPerCell, ebiten.KeyD))
	if p.World().Dwarf.X != x+config.CellSize {
		t.Errorf("x %d, want %d", p.World().Dwarf.X, x+config.CellSize)
	}
}

func TestEscPauses(t *testing.T) {
	h, p := start(t)
	x := p.World().Dwarf.X
	h.Press(ebiten.KeyEscape)
	scenetest.Current[*scene.Pause](h)
	h.Run(scenetest.Hold(ticksPerCell, ebiten.KeyArrowRight))
	if p.World().Dwarf.X != x {
		t.Error("moved while paused")
	}
}

// TestGoldenReplay digs a fixed route and checks the result as recorded in
// testdata. After an intended change to the rules, run it with -update.
func TestGoldenReplay(t *testing.T) {
	h, p := start(t)
	walk(h, p, ebiten.KeyArrowRight, 3)
	walk(h, p, ebiten.KeyArrowUp, 3)
	walk(h, p, ebiten.KeyArrowLeft, 5)
	walk(h, p, ebiten.KeyArrowDown, 5)
	walk(h, p, ebiten.KeyArrowLeft, 1) // onto a jewel
	got := fmt.Sprintf("score %d  jewels left %d  level %d\ndwarf (%d, %d)\n%v",
		h.Ctx.Session.Score, p.World().JewelsLeft, p.Level(), p.World().Dwarf.X, p.World().Dwarf.Y, p.World())
	scenetest.Golden(t, "replay", got)
}

func TestSounds(t *testing.T) {
	if err := Sounds.Check(); err != nil {
		t.Error(err)
	}
}

// TestSprites checks the sprite source parses and packs, and has every sprite
// the scene draws, a whole cell in size.
func TestSprites(t *testing.T) {
	f, err := os.Open("../../assets/sprites/gemburrow.txt")
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
	size := map[string]int{}
	for _, s := range sprites {
		if s.W == s.H {
			size[s.Name] = s.W
		}
	}
	for _, name := range []string{"earth", "tunnel", "jewel", "dwarf"} {
		if size[name] != config.CellSize {
			t.Errorf("sprite %q is not %d square", name, config.CellSize)
		}
	}
}
