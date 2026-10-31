package play

import (
	"testing"

	"github.com/comprofix/gemburrow/internal/config"
)

func actorOf(t *testing.T, actors []titleActor, kind titleKind) titleActor {
	t.Helper()
	for _, a := range actors {
		if a.kind == kind {
			return a
		}
	}
	t.Fatalf("no actor of kind %d", kind)
	return titleActor{}
}

func TestTheTitleDwarfDigsTheTunnelAsTheyGo(t *testing.T) {
	cols := config.Modern.ScreenWidth / config.CellSize
	dug, _ := titleScene(0)
	if dug != 0 {
		t.Errorf("dug %d before the dwarf arrives", dug)
	}
	last := 0
	for ticks := range int64(config.TitlePassTicks) {
		dug, actors := titleScene(ticks)
		if dug < last {
			t.Fatalf("tick %d: the tunnel shrank from %d to %d", ticks, last, dug)
		}
		last = dug
		d := actorOf(t, actors, titleDwarf)
		if d.x >= 0 && d.x+config.CellSize/2 < config.Modern.ScreenWidth && (d.x+config.CellSize/2)/config.CellSize >= dug {
			t.Fatalf("tick %d: dwarf at %d outside the %d cells dug", ticks, d.x, dug)
		}
		if g := actorOf(t, actors, titleKobold); g.x >= d.x {
			t.Fatalf("tick %d: kobold at %d is not behind the dwarf at %d", ticks, g.x, d.x)
		}
	}
	if last != cols {
		t.Errorf("the tunnel ended at %d cells, want %d", last, cols)
	}
}

func TestTheTitleChaseTurnsAroundAndAllLeaveTheScreen(t *testing.T) {
	w := config.Modern.ScreenWidth
	_, back := titleScene(config.TitlePassTicks)
	if d := actorOf(t, back, titleDwarf); !d.left || d.scared {
		t.Errorf("dwarf on the way back: %+v", d)
	}
	if g := actorOf(t, back, titleKobold); !g.left || !g.scared {
		t.Errorf("kobold on the way back: %+v", g)
	}
	_, first := titleScene(config.TitlePassTicks - 1)
	for _, a := range first {
		if a.x < w {
			t.Errorf("still on screen as the first pass ends: %+v", a)
		}
	}
	_, second := titleScene(2*config.TitlePassTicks - 1)
	for _, a := range second {
		if a.x+config.CellSize > 0 {
			t.Errorf("still on screen as the second pass ends: %+v", a)
		}
	}
	dug, third := titleScene(2 * config.TitlePassTicks)
	if dug != w/config.CellSize || actorOf(t, third, titleDwarf).dig {
		t.Errorf("third pass: dug %d, dwarf digging %v", dug, actorOf(t, third, titleDwarf).dig)
	}
}

func TestTheTitleLogoAndCavernFitTheScreen(t *testing.T) {
	name := "GEM BURROW"
	for _, r := range name {
		if _, ok := titleLetters[r]; !ok && r != ' ' {
			t.Errorf("no letter %q", r)
		}
	}
	width := (len(name)*(config.TitleLetterCols+1) - 1) * config.TitleBlockPixels
	if width > config.Modern.ScreenWidth {
		t.Errorf("logo is %d wide", width)
	}
	if bottom := config.TitleLogoY + config.TitleLetterRows*config.TitleBlockPixels; bottom > config.TitleSceneY {
		t.Errorf("logo ends at %d, under the cavern at %d", bottom, config.TitleSceneY)
	}
	for _, c := range append(append(append([][2]int{}, titleJewels...), titleSacks...), titleGold...) {
		if c[1] == config.TitleTunnelRow || c[1] >= config.TitleRows {
			t.Errorf("item at %v is in the tunnel or outside the cavern", c)
		}
	}
}
