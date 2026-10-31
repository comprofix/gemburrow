package play

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gemburrow/internal/config"
	"github.com/comprofix/gemburrow/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/gemburrow/internal/input"
	"github.com/comprofix/gemburrow/internal/scene"
	"github.com/comprofix/gemburrow/internal/scene/scenetest"
	"github.com/comprofix/gemburrow/internal/spritegrid"
	"github.com/comprofix/gemburrow/internal/world"
)

const ticksPerCell = config.CellSize / config.DwarfSpeed

func start(t *testing.T) (*scenetest.Harness, *Play) {
	h := scenetest.New(t, 42)
	h.Ctx.NewPlay = New
	h.Ctx.Input = input.New(h.Script, Bindings...)
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
	// This test is about the digging: no goblins, and no sacks in the way.
	rows := slices.Clone(config.LevelFor(1))
	for i, r := range rows {
		rows[i] = strings.ReplaceAll(r, string(rune(config.LevelSack)), string(rune(config.LevelEarth)))
	}
	l, err := world.Parse(rows)
	if err != nil {
		t.Fatal(err)
	}
	tuning := config.BaseTuning
	tuning.GoblinTotal = 0
	p.world = world.New(h.Ctx.Session, l, tuning, h.Ctx.RNG)
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
	scenetest.Golden(t, "replay", summary(h, p))
}

// summary is the game's state, for the golden files.
func summary(h *scenetest.Harness, p *Play) string {
	w := p.World()
	return fmt.Sprintf("score %d  lives %d  jewels left %d  level %d\ndwarf (%d, %d)  caught %v  over %v\ngoblins %v  entered %d  killed %d\n%v",
		h.Ctx.Session.Score, h.Ctx.Session.Lives, w.JewelsLeft, p.Level(), w.Dwarf.X, w.Dwarf.Y, w.Caught, w.Over, w.Goblins, w.Spawned, w.Killed, w)
}

// TestGoldenSackDrop digs out under a sack with a goblin coming along the
// tunnel below, gets clear, lets the sack crush the goblin and burst, then
// collects the gold.
func TestGoldenSackDrop(t *testing.T) {
	h, p := start(t)
	rows := []string{
		"###############",
		"###############",
		"###############",
		"###############",
		"#####$#########",
		"###############",
		"###############",
		"=====D=========",
		"###############",
		"####*#########G",
	}
	l, err := world.Parse(rows)
	if err != nil {
		t.Fatal(err)
	}
	tuning := config.BaseTuning
	tuning.GoblinTotal, tuning.TrollAfterTicks = 2, -1
	tuning.GoblinSpawnTicks = 100 * config.TPS
	p.world = world.New(h.Ctx.Session, l, tuning, h.Ctx.RNG)
	w := p.World()
	w.Goblins = []world.Goblin{{X: 1 * config.CellSize, Y: 7 * config.CellSize}}
	w.Spawned = 1

	walk(h, p, ebiten.KeyArrowUp, 2)
	walk(h, p, ebiten.KeyArrowDown, 2)
	walk(h, p, ebiten.KeyArrowRight, 3)
	for len(w.Sacks) > 0 {
		h.Ticks(1)
	}
	if w.Killed != 1 || len(w.Gold) != 1 {
		t.Fatalf("killed %d, gold %v:\n%v", w.Killed, w.Gold, w)
	}
	walk(h, p, ebiten.KeyArrowLeft, 3)
	if h.Ctx.Session.Score != config.GoblinPoints+config.GoldPoints {
		t.Errorf("score %d", h.Ctx.Session.Score)
	}
	scenetest.Golden(t, "sackdrop", summary(h, p))
}

func TestLoopingTunesKeepTheirPartsInStep(t *testing.T) {
	for _, tune := range Sounds.Tunes {
		if !tune.Loop {
			continue
		}
		var first time.Duration
		for i, part := range tune.Parts {
			var sum time.Duration
			for _, n := range part.Notes {
				sum += n.Dur
			}
			if i == 0 {
				first = sum
			} else if d := sum - first; d > time.Millisecond || d < -time.Millisecond {
				t.Errorf("%s: part %d lasts %v, part 0 %v", tune.Name, i, sum, first)
			}
		}
	}
}

// TestAGameRunsThroughEveryLevelAtEachSkill goes on past the last layout,
// checking each level's layout and tuning.
func TestAGameRunsThroughEveryLevelAtEachSkill(t *testing.T) {
	for _, skill := range config.Skills {
		h := scenetest.New(t, 42)
		h.Ctx.NewPlay = New
		h.Ctx.Input = input.New(h.Script, Bindings...)
		p := New(h.Ctx, skill).(*Play)
		h.Start(p)
		h.Ticks(scene.FadeTicks)
		for level := 1; level <= len(config.Levels)+config.RampLevels; level++ {
			if p.Level() != level {
				t.Fatalf("%s: on level %d, want %d", skill.ID, p.Level(), level)
			}
			if got, want := p.World().Tuning, config.TuningFor(skill, level); got != want {
				t.Errorf("%s level %d: tuning %+v, want %+v", skill.ID, level, got, want)
			}
			l, err := world.Parse(config.LevelFor(level))
			if err != nil {
				t.Fatal(err)
			}
			if want := world.New(h.Ctx.Session, l, p.World().Tuning, h.Ctx.RNG); want.String() != p.World().String() {
				t.Errorf("%s level %d: wrong layout:\n%v", skill.ID, level, p.World())
			}
			p.World().JewelsLeft = 0
			h.Ticks(config.LevelClearTicks + 2)
		}
	}
}

// TestGoldenSackPush pushes a sack of the first level over a jewel, which it
// leaves in place for the dwarf to take, and on through the earth.
func TestGoldenSackPush(t *testing.T) {
	h, p := start(t)
	w := p.World()
	w.Tuning.GoblinTotal = 0
	walk(h, p, ebiten.KeyArrowLeft, 5)
	walk(h, p, ebiten.KeyArrowDown, 1)
	// Pushing is at half walking speed, so this is two cells.
	h.Run(scenetest.Hold(4*ticksPerCell, ebiten.KeyArrowLeft))
	if s := w.Sacks[2]; s.X != 1*config.CellSize || !w.Earth(1, 5) || h.Ctx.Session.Score != config.JewelPoints {
		t.Errorf("sack %+v, earth under it %v, score %d", s, w.Earth(1, 5), h.Ctx.Session.Score)
	}
	scenetest.Golden(t, "sackpush", fmt.Sprintf("%s\nsacks %+v\n", summary(h, p), w.Sacks))
}

// TestGoldenChase walks the open tunnel, fires, then stands still until the
// goblins have taken every life.
func TestGoldenChase(t *testing.T) {
	h, p := start(t)
	walk(h, p, ebiten.KeyArrowRight, 3)
	h.Run(scenetest.Hold(1, ebiten.KeySpace))
	for !p.World().Over {
		h.Ticks(1)
	}
	scenetest.Golden(t, "chase", summary(h, p))
}

func TestSpaceFires(t *testing.T) {
	h, p := start(t)
	h.Run(scenetest.Hold(1, ebiten.KeySpace), scenetest.Hold(1))
	w := p.World()
	if w.Shot == nil || w.Recharge == 0 {
		t.Errorf("shot %v, recharge %d", w.Shot, w.Recharge)
	}
}

func TestCaughtRespawnsAfterAPause(t *testing.T) {
	h, p := start(t)
	w := p.World()
	w.Goblins = append(w.Goblins, world.Goblin{X: w.Dwarf.X, Y: w.Dwarf.Y})
	h.Ticks(1)
	if !w.Caught || h.Ctx.Session.Lives != config.StartLives-1 {
		t.Fatalf("caught %v, lives %d", w.Caught, h.Ctx.Session.Lives)
	}
	h.Ticks(config.CaughtTicks)
	if w.Caught {
		t.Error("still caught after the pause")
	}
}

// TestAWholeGame loses every life with a score, then goes through initials
// to the high score table.
func TestAWholeGame(t *testing.T) {
	h, p := start(t)
	skill := h.Ctx.Session.Skill
	h.Ctx.Session.Score = 1000
	for !p.World().Over {
		h.Ticks(1)
	}
	if lives := h.Ctx.Session.Lives; lives != 0 {
		t.Errorf("lives %d", lives)
	}
	h.Ticks(config.GameOverTicks + 2*scene.FadeTicks)
	scenetest.Current[*scene.HighScoreEntry](h)
	for range config.InitialsLen {
		h.Press(ebiten.KeyEnter)
	}
	scenetest.Current[*scene.HighScoreTable](h)
	if es := h.Ctx.Scores.Table(skill.ID); len(es) != 1 || es[0].Score != 1000 {
		t.Errorf("table %v", es)
	}
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
	if size["shot"] != config.ShotSize {
		t.Errorf("sprite %q is not %d square", "shot", config.ShotSize)
	}
	for _, name := range []string{"earth", "tunnel", "jewel", "dwarf", "goblin", "goblin_digger", "sack", "gold"} {
		if size[name] != config.CellSize {
			t.Errorf("sprite %q is not %d square", name, config.CellSize)
		}
	}
}
