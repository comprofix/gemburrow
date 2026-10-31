package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/comprofix/gameskeleton/internal/gfx/palettes/resurrect64"
)

// one is a game with the single look p.
func one(p Preset) Looks { return Looks{First: p} }

// two is a game with looks a and b, a first.
func two(a, b Preset) Looks { return Looks{First: a, Second: &b} }

// pixel320 is a 320×200 pixel look sharing Smooth's palette and sound, to
// pair with a 1280×800 smooth look at four times.
var pixel320 = Preset{
	Name: "Pixel", ScreenWidth: 320, ScreenHeight: 200,
	ScaleModes: []ScaleMode{ScaleInteger, ScaleCRT},
	Palette:    resurrect64.Palette, Audio: Chip4, Art: ArtPixel,
}

func TestLooksCheck(t *testing.T) {
	smooth800 := Smooth
	smooth800.Name, smooth800.ScreenHeight = "Smooth 800", 800
	for _, c := range []struct {
		name  string
		looks Looks
		bad   string // part of the error, or "" for none
	}{
		{"one look", one(Classic), ""},
		{"smooth then pixel", two(Smooth, Modern), ""},
		{"pixel then smooth", two(Modern, Smooth), ""},
		{"four times", two(smooth800, pixel320), ""},
		{"same name", two(Modern, Modern), "named"},
		{"both pixel", two(Modern, pixel320), "both pixel"},
		{"palettes", two(Smooth, Classic), "palettes"},
		{"not a multiple", two(Smooth, pixel320), "multiple"},
	} {
		err := c.looks.Check()
		switch {
		case c.bad == "" && err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.bad != "" && (err == nil || !strings.Contains(err.Error(), c.bad)):
			t.Errorf("%s: error %v, want one about %q", c.name, err, c.bad)
		}
	}
	if err := ProjectLooks().Check(); err != nil {
		t.Errorf("project.go's looks: %v", err)
	}
}

func TestPixelLook(t *testing.T) {
	if p := one(Modern).Pixel(); p != nil {
		t.Errorf("one look: pixel look %s, want none", p.Name)
	}
	for _, l := range []Looks{two(Smooth, Modern), two(Modern, Smooth)} {
		if p := l.Pixel(); p == nil || p.Name != Modern.Name {
			t.Errorf("%s and %s: pixel look %v, want Modern", l.First.Name, l.Second.Name, p)
		}
	}
}

func TestTwoLookSettings(t *testing.T) {
	l := two(Smooth, Modern)
	d := DefaultSettings(l)
	if d.Look != Smooth.Name || d.ScaleMode != ScaleInteger || d.Mode(l) != ScaleSmooth {
		t.Errorf("defaults: look %q, mode %q shown as %q; want Smooth, the pixel look's integer, shown smooth", d.Look, d.ScaleMode, d.Mode(l))
	}
	if s := (Settings{Look: Modern.Name, ScaleMode: ScaleInteger}).Clamped(l); s.Look != Modern.Name || s.Shown(l).Name != Modern.Name || s.Mode(l) != ScaleInteger {
		t.Errorf("the second look: %+v", s)
	}
	if s := (Settings{Look: "Bogus"}).Clamped(l); s.Look != Smooth.Name {
		t.Errorf("an unknown look became %q, want the first", s.Look)
	}

	// The pixel look's CRT is kept while the smooth look is shown.
	crt := two(Smooth, pixel320)
	crt.First.ScreenWidth, crt.First.ScreenHeight = 1280, 800
	if s := (Settings{Look: Smooth.Name, ScaleMode: ScaleCRT}).Clamped(crt); s.ScaleMode != ScaleCRT || s.Mode(crt) != ScaleSmooth {
		t.Errorf("CRT under the smooth look: saved %q, shown %q", s.ScaleMode, s.Mode(crt))
	}

	// A file from before the look was saved loads on the first look.
	path := filepath.Join(t.TempDir(), SettingsFile)
	os.WriteFile(path, []byte(`{"scale_mode": "integer", "fullscreen": true}`), 0o644)
	if s := LoadSettings(path, l); s.Look != Smooth.Name || !s.Fullscreen {
		t.Errorf("an old file loaded as %+v", s)
	}
}

// TestOneLookSettingsFile checks a game with one look neither saves the
// look nor keeps one read from a file.
func TestOneLookSettingsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), SettingsFile)
	if err := SaveSettings(path, DefaultSettings(one(Modern))); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), `"look"`) {
		t.Errorf("a one-look game saved the look:\n%s", data)
	}
	os.WriteFile(path, []byte(`{"look": "Smooth"}`), 0o644)
	if s := LoadSettings(path, one(Modern)); s.Look != "" {
		t.Errorf("a one-look game kept the look %q", s.Look)
	}
}
