package bitfont

import (
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
)

const tiny = `; a 4×4 test font
@A
.##.
#..#
####
#..#
@space
....
....
....
....
@/
...#
..#.
.#..
#...
`

func TestParse(t *testing.T) {
	f, err := Parse(strings.NewReader(tiny))
	if err != nil {
		t.Fatal(err)
	}
	if f.Size != 4 {
		t.Errorf("size %d, want 4", f.Size)
	}
	if got := string(f.Runes()); got != " /A" {
		t.Errorf("runes %q", got)
	}
	if !f.Ink('A', 1, 0) || f.Ink('A', 0, 0) || !f.Ink('A', 3, 3) {
		t.Error("glyph A pixels wrong")
	}
	if f.Ink('A', 4, 0) || f.Ink('A', -1, 0) || f.Ink('Z', 0, 0) {
		t.Error("pixels outside a glyph or of a missing glyph must be empty")
	}
	if f.Width("AA/") != 12 {
		t.Errorf("width %d, want 12", f.Width("AA/"))
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"bad header":     "A\n##\n##\n",
		"bad pixel":      "@A\n#x\n##\n",
		"wrong width":    "@A\n##\n###\n",
		"incomplete":     "@A\n##\n",
		"duplicate":      "@A\n##\n##\n@A\n##\n##\n",
		"empty":          "; nothing\n",
		"too wide":       "@A\n" + strings.Repeat("#", MaxSize+1) + "\n",
		"two-char glyph": "@AB\n##\n##\n",
	}
	for name, src := range cases {
		if _, err := Parse(strings.NewReader(src)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestDraw(t *testing.T) {
	f, _ := Parse(strings.NewReader(tiny))
	dst := image.NewRGBA(image.Rect(0, 0, 10, 4))
	ink := color.RGBA{0xff, 0, 0, 0xff}
	f.Draw(dst, "A?", 1, 0, ink) // '?' is missing and falls back to '?', also missing: drawn empty
	if dst.RGBAAt(2, 0) != ink || dst.RGBAAt(1, 0) == ink {
		t.Error("A drawn in the wrong place")
	}
	for x := 5; x < 9; x++ {
		for y := range 4 {
			if dst.RGBAAt(x, y) == ink {
				t.Fatal("a character missing from the font drew ink")
			}
		}
	}
}

func TestScale2x(t *testing.T) {
	f, _ := Parse(strings.NewReader(tiny))
	big := Scale2x(f)
	if big.Size != 8 {
		t.Fatalf("size %d, want 8", big.Size)
	}
	// A solid run stays solid: A's bar (row 2) doubles to rows 4–5.
	for x := range 8 {
		if !big.Ink('A', x, 4) || !big.Ink('A', x, 5) {
			t.Errorf("A's bar has a gap at x=%d", x)
		}
	}
	// A diagonal is smoothed: each step of '/' gains in-between pixels, so
	// the doubled line is a continuous two-pixel-wide stroke with no
	// staircase corners left empty.
	want := []string{
		"......##",
		".....###",
		"....###.",
		"...###..",
		"..###...",
		".###....",
		"###.....",
		"##......",
	}
	for y, row := range want {
		for x, ch := range row {
			if big.Ink('/', x, y) != (ch == '#') {
				t.Errorf("'/' at (%d,%d): got %v", x, y, big.Ink('/', x, y))
			}
		}
	}
}

func TestShippedFont(t *testing.T) {
	data, err := os.Open("../../assets/font/font8x8.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()
	f, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if f.Size != 8 {
		t.Errorf("size %d, want 8", f.Size)
	}
	for r := ' '; r <= '~'; r++ {
		if !f.Has(r) {
			t.Errorf("missing printable ASCII %q", r)
		}
	}
}
