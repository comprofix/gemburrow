package spritegrid

import (
	"image/color"
	"strings"
	"testing"

	"github.com/comprofix/gameskeleton/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/gameskeleton/internal/gfx/palettes/sweetie16"
)

const hexSrc = `; hex-digit sprites (Sweetie 16)
sprite box 3 2
0C0
.C.

C0C
.0.
recolour box_red from box C=2
`

func TestParseHex(t *testing.T) {
	sprites, err := Parse(strings.NewReader(hexSrc), sweetie16.Palette)
	if err != nil {
		t.Fatal(err)
	}
	if len(sprites) != 2 {
		t.Fatalf("%d sprites, want 2", len(sprites))
	}
	box := sprites[0]
	if box.Name != "box" || box.W != 3 || box.H != 2 || len(box.Frames) != 2 {
		t.Fatalf("box = %+v", box)
	}
	want0 := []int{0, 12, 0, Transparent, 12, Transparent}
	for i, v := range want0 {
		if box.Frames[0][i] != v {
			t.Errorf("frame 0 pixel %d = %d, want %d", i, box.Frames[0][i], v)
		}
	}
	red := sprites[1]
	if red.Name != "box_red" || red.Frames[0][1] != sweetie16.Red || red.Frames[0][0] != 0 || red.Frames[1][0] != sweetie16.Red {
		t.Errorf("recolour gave %+v", red)
	}
}

const keySrc = `colours k=Ink g=Moss y=Gold
sprite leaf 2 2
kg
.y
recolour leaf_gold from leaf g=y
`

func TestParseColourKey(t *testing.T) {
	sprites, err := Parse(strings.NewReader(keySrc), resurrect64.Palette)
	if err != nil {
		t.Fatal(err)
	}
	leaf, gold := sprites[0], sprites[1]
	want := []int{resurrect64.Ink, resurrect64.Moss, Transparent, resurrect64.Gold}
	for i, v := range want {
		if leaf.Frames[0][i] != v {
			t.Errorf("leaf pixel %d = %d, want %d", i, leaf.Frames[0][i], v)
		}
	}
	if gold.Frames[0][1] != resurrect64.Gold || gold.Frames[0][0] != resurrect64.Ink {
		t.Errorf("leaf_gold = %v", gold.Frames[0])
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name, src, wantMsg string
	}{
		{"pixels before header", "00\n", "line 1: pixels before any sprite header"},
		{"bad header", "sprite a 2\n", "line 1: want \"sprite NAME"},
		{"extra header field", "sprite a 1 1 9\n0\n", "line 1: want \"sprite NAME"},
		{"zero size", "sprite a 0 1\n", "sizes must be"},
		{"too big", "sprite a 2000 1\n", "sizes must be"},
		{"row too wide", "sprite a 2 1\n000\n", "line 2: sprite \"a\" row is 3 wide, want 2"},
		{"too few rows", "sprite a 1 2\n0\n\n", "line 3: sprite \"a\" frame 0 has 1 rows, want 2"},
		{"no frames", "sprite a 1 1\n", "sprite \"a\" has no frames"},
		{"bad hex pixel", "sprite a 1 1\nz\n", "bad pixel 'z'"},
		{"duplicate sprite", "sprite a 1 1\n0\nsprite a 1 1\n0\n", "line 3: sprite \"a\" is already defined on line 1"},
		{"recolour duplicate", "sprite a 1 1\n0\nrecolour a from a 0=1\n", "already defined"},
		{"recolour missing source", "sprite a 1 1\n0\nrecolour b from c 0=1\n", "no sprite \"c\""},
		{"recolour bad syntax", "sprite a 1 1\n0\nrecolour b of a 0=1\n", "want \"recolour NEW"},
		{"recolour bad swap", "sprite a 1 1\n0\nrecolour b from a 0-1\n", "bad colour swap"},
		{"key after sprite", "sprite a 1 1\n0\ncolours k=Ink\n", "must come before any sprite"},
		{"two keys", "colours k=Ink\ncolours g=Moss\n", "only one colours line"},
		{"empty key", "colours\n", "want \"colours"},
		{"bad key entry", "colours kInk\n", "bad colour key entry"},
		{"dot as key", "colours .=Ink\n", "cannot be a key character"},
		{"key char twice", "colours k=Ink k=Moss\n", "used twice"},
		{"unknown colour name", "colours k=Inky\n", "has no colour named \"Inky\""},
		{"pixel not in key", "colours k=Ink\nsprite a 2 1\nkz\n", "'z' is not in the colours key"},
		{"swap not in key", "colours k=Ink\nsprite a 1 1\nk\nrecolour b from a k=z\n", "not in the colours key"},
	}
	for _, c := range cases {
		_, err := Parse(strings.NewReader(c.src), resurrect64.Palette)
		if err == nil {
			t.Errorf("%s: no error", c.name)
			continue
		}
		if !strings.Contains(err.Error(), c.wantMsg) {
			t.Errorf("%s: error %q, want it to contain %q", c.name, err, c.wantMsg)
		}
	}
}

func TestHexOutsideSmallPalette(t *testing.T) {
	small := sweetie16.Palette
	small.Colors, small.Names = small.Colors[:4], small.Names[:4]
	_, err := Parse(strings.NewReader("sprite a 1 1\n9\n"), small)
	if err == nil || !strings.Contains(err.Error(), "outside palette") {
		t.Errorf("got %v, want an outside-palette error", err)
	}
}

func TestPack(t *testing.T) {
	sprites, err := Parse(strings.NewReader(hexSrc), sweetie16.Palette)
	if err != nil {
		t.Fatal(err)
	}
	img, index, err := Pack(sprites, sweetie16.Palette)
	if err != nil {
		t.Fatal(err)
	}
	box, red := index["box"], index["box_red"]
	if box != (Entry{X: 0, Y: 0, W: 3, H: 2, Frames: 2}) || red != (Entry{X: 0, Y: 2, W: 3, H: 2, Frames: 2}) {
		t.Errorf("index %+v", index)
	}
	white := sweetie16.Palette.Colors[sweetie16.White]
	if img.RGBAAt(1, 0) != white { // box frame 0, middle of the top row
		t.Error("box pixel missing")
	}
	if img.RGBAAt(3, 0) != white { // box frame 1 starts at x=3
		t.Error("frame 1 not placed to the right of frame 0")
	}
	if img.RGBAAt(0, 1) != (color.RGBA{}) {
		t.Error("a transparent pixel was filled")
	}
	if img.RGBAAt(1, 2) != sweetie16.Palette.Colors[sweetie16.Red] {
		t.Error("box_red not packed below box")
	}

	if _, _, err := Pack([]Sprite{{Name: "a", W: 1, H: 1, Frames: [][]int{{0}}}, {Name: "a", W: 1, H: 1, Frames: [][]int{{0}}}}, sweetie16.Palette); err == nil {
		t.Error("duplicate names not rejected")
	}
	if _, _, err := Pack([]Sprite{{Name: "a", W: 1, H: 1, Frames: [][]int{{99}}}}, sweetie16.Palette); err == nil {
		t.Error("colour outside the palette not rejected")
	}
}
