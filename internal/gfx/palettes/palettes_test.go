package palettes_test

import (
	"image/color"
	"testing"

	"github.com/comprofix/gameskeleton/internal/gfx/palettes"
	"github.com/comprofix/gameskeleton/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/gameskeleton/internal/gfx/palettes/sweetie16"
)

func TestReadyMadePalettes(t *testing.T) {
	cases := []struct {
		p                  palettes.Palette
		size               int
		darkest, brightest int
		checkName          string
		checkIndex         int
		checkColour        color.RGBA
	}{
		{sweetie16.Palette, 16, sweetie16.Night, sweetie16.White, "Sky", sweetie16.Sky, color.RGBA{0x41, 0xa6, 0xf6, 0xff}},
		{resurrect64.Palette, 64, resurrect64.Ink, resurrect64.White, "Gold", resurrect64.Gold, color.RGBA{0xf9, 0xc2, 0x2b, 0xff}},
	}
	for _, c := range cases {
		t.Run(c.p.Name, func(t *testing.T) {
			if c.p.Len() != c.size || len(c.p.Names) != c.size {
				t.Fatalf("%d colours, %d names; want %d", c.p.Len(), len(c.p.Names), c.size)
			}
			names := map[string]bool{}
			colours := map[color.RGBA]bool{}
			for i, n := range c.p.Names {
				if n == "" {
					t.Errorf("colour %d has no name", i)
				}
				if names[n] {
					t.Errorf("duplicate name %q", n)
				}
				names[n] = true
				col := c.p.Colors[i]
				if colours[col] {
					t.Errorf("duplicate colour %v (%s)", col, n)
				}
				colours[col] = true
				if col.A != 0xff {
					t.Errorf("%s is not opaque", n)
				}
				if got, ok := c.p.Lookup(n); !ok || got != i {
					t.Errorf("Lookup(%q) = %d, %v; want %d", n, got, ok, i)
				}
			}
			if got, ok := c.p.Lookup(c.checkName); !ok || got != c.checkIndex || c.p.Colors[got] != c.checkColour {
				t.Errorf("Lookup(%q) gave %d (%v), want %d (%v)", c.checkName, got, c.p.Colors[got], c.checkIndex, c.checkColour)
			}
			if _, ok := c.p.Lookup("NoSuchColour"); ok {
				t.Error("Lookup found a colour that doesn't exist")
			}
			if got := c.p.Darkest(); got != c.darkest {
				t.Errorf("Darkest = %s, want %s", c.p.Names[got], c.p.Names[c.darkest])
			}
			if got := c.p.Brightest(); got != c.brightest {
				t.Errorf("Brightest = %s, want %s", c.p.Names[got], c.p.Names[c.brightest])
			}
		})
	}
}

func TestNew(t *testing.T) {
	p := palettes.New("Test", []palettes.Entry{{Name: "A", Hex: 0x102030}, {Name: "B", Hex: 0xff00ee}})
	want := []color.RGBA{{0x10, 0x20, 0x30, 0xff}, {0xff, 0x00, 0xee, 0xff}}
	for i, w := range want {
		if p.Colors[i] != w {
			t.Errorf("colour %d = %v, want %v", i, p.Colors[i], w)
		}
	}
}
