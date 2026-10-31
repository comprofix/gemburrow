package config

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

var presets = []Preset{Modern, Classic, Smooth}

func TestPresets(t *testing.T) {
	cases := []struct {
		p          Preset
		w, h       int
		defW, defH int
		colours    int
	}{
		{Modern, 640, 360, 1280, 720, 64},
		{Classic, 320, 200, 1280, 800, 16},
		{Smooth, 1280, 720, 1280, 720, 64},
	}
	for _, c := range cases {
		t.Run(c.p.Name, func(t *testing.T) {
			if c.p.ScreenWidth != c.w || c.p.ScreenHeight != c.h {
				t.Errorf("screen %d×%d, want %d×%d", c.p.ScreenWidth, c.p.ScreenHeight, c.w, c.h)
			}
			if w, h := c.p.DefaultWindow(); w != c.defW || h != c.defH {
				t.Errorf("default window %d×%d, want %d×%d", w, h, c.defW, c.defH)
			}
			if c.p.Palette.Len() != c.colours {
				t.Errorf("%d colours, want %d", c.p.Palette.Len(), c.colours)
			}
			first := ScaleInteger
			if c.p.Art == ArtSmooth {
				first = ScaleSmooth
			}
			if len(c.p.ScaleModes) == 0 || c.p.ScaleModes[0] != first {
				t.Errorf("scale modes %v: want %q first", c.p.ScaleModes, first)
			}
		})
	}
}

func TestMinWindow(t *testing.T) {
	for _, c := range []struct {
		p    Preset
		w, h int
	}{{Modern, 640, 360}, {Classic, 320, 200}, {Smooth, 640, 360}} {
		if w, h := c.p.MinWindow(); w != c.w || h != c.h {
			t.Errorf("%s: min window %d×%d, want %d×%d", c.p.Name, w, h, c.w, c.h)
		}
	}
}

func TestNextMode(t *testing.T) {
	if got := Modern.NextMode(ScaleInteger); got != ScaleInteger {
		t.Errorf("Modern has one mode; next = %q", got)
	}
	if got := Classic.NextMode(ScaleInteger); got != ScaleCRT {
		t.Errorf("Classic integer → %q, want crt", got)
	}
	if got := Classic.NextMode(ScaleCRT); got != ScaleInteger {
		t.Errorf("Classic crt → %q, want integer", got)
	}
	if got := Classic.NextMode("bogus"); got != ScaleInteger {
		t.Errorf("unknown mode → %q, want the first mode", got)
	}
}

func TestLoadSettingsMissingOrCorrupt(t *testing.T) {
	dir := t.TempDir()
	for _, p := range presets {
		if got := LoadSettings(filepath.Join(dir, "missing.json"), one(p)); got != DefaultSettings(one(p)) {
			t.Errorf("%s missing file: %+v", p.Name, got)
		}
		bad := filepath.Join(dir, "bad.json")
		os.WriteFile(bad, []byte("{not json"), 0o644)
		if got := LoadSettings(bad, one(p)); got != DefaultSettings(one(p)) {
			t.Errorf("%s corrupt file: %+v", p.Name, got)
		}
	}
}

func TestLoadSettingsKeepsDefaultsForMissingFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), SettingsFile)
	os.WriteFile(path, []byte(`{"fullscreen": true}`), 0o644)
	got := LoadSettings(path, one(Modern))
	want := DefaultSettings(one(Modern))
	want.Fullscreen = true
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestClamped(t *testing.T) {
	s := Settings{
		ScaleMode: ScaleCRT, // Modern doesn't offer CRT
		WindowW:   10, WindowH: 99999,
		MasterVolume: -1, MusicVolume: 2, SFXVolume: math.NaN(),
	}.Clamped(one(Modern))
	if s.ScaleMode != ScaleInteger {
		t.Errorf("scale mode %q, want integer", s.ScaleMode)
	}
	if s.WindowW != 640 || s.WindowH != MaxWindowH {
		t.Errorf("window %d×%d, want 640×%d", s.WindowW, s.WindowH, MaxWindowH)
	}
	if s.MasterVolume != 0 || s.MusicVolume != 1 || s.SFXVolume != 0 {
		t.Errorf("volumes %v %v %v, want 0 1 0", s.MasterVolume, s.MusicVolume, s.SFXVolume)
	}
	if got := (Settings{ScaleMode: ScaleCRT, WindowW: 640, WindowH: 400}).Clamped(one(Classic)); got.ScaleMode != ScaleCRT {
		t.Errorf("Classic should keep CRT, got %q", got.ScaleMode)
	}
}

func TestSaveAndLoadSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, SettingsFile)
	want := DefaultSettings(one(Classic))
	want.ScaleMode = ScaleCRT
	want.Fullscreen = true
	want.WindowW, want.WindowH = 1600, 1000
	want.MusicVolume = 0.25
	want.ReducedMotion = true
	if err := SaveSettings(path, want); err != nil {
		t.Fatal(err)
	}
	if got := LoadSettings(path, one(Classic)); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	// Saving again replaces the file and leaves no temporary files behind.
	want.SoundOn = false
	if err := SaveSettings(path, want); err != nil {
		t.Fatal(err)
	}
	if got := LoadSettings(path, one(Classic)); got != want {
		t.Errorf("after resave got %+v, want %+v", got, want)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("want only %s in the folder, found %d entries", SettingsFile, len(entries))
	}
}

func TestInGoBuild(t *testing.T) {
	// Paths in this OS's own form: on Linux a backslash is not a separator.
	if !inGoBuild(filepath.Join("tmp", "go-build123", "b001", "exe")) {
		t.Error("go-build folder not detected")
	}
	if inGoBuild(filepath.Join("games", "gameskeleton")) {
		t.Error("ordinary folder taken for go-build")
	}
}

func TestAudioLevels(t *testing.T) {
	s := DefaultSettings(one(Modern))
	s.MasterVolume, s.MusicVolume, s.SFXVolume = 0.9, 0.6, 0.3
	type levels struct{ master, music, sfx float64 }
	get := func(s Settings) levels {
		m, mu, fx := s.AudioLevels()
		return levels{m, mu, fx}
	}
	for _, c := range []struct {
		name               string
		sound, music, sfxs bool
		want               levels
	}{
		{"all on", true, true, true, levels{0.9, 0.6, 0.3}},
		{"music off", true, false, true, levels{0.9, 0, 0.3}},
		{"effects off", true, true, false, levels{0.9, 0.6, 0}},
		{"both off", true, false, false, levels{0.9, 0, 0}},
		{"sound off", false, true, true, levels{0, 0, 0}},
	} {
		s.SoundOn, s.MusicOn, s.SFXOn = c.sound, c.music, c.sfxs
		if got := get(s); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
