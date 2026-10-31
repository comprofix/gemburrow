package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// The saved files, kept beside the game.
const (
	SettingsFile = "settings.json"
	ScoresFile   = "scores.json"
)

// Settings are the player's choices saved between runs. They
// cover presentation and sound only and never change the rules.
type Settings struct {
	// Look is the name of the look shown, in a game with two (see Looks);
	// empty, and not saved, in a game with one.
	Look string `json:"look,omitempty"`
	// ScaleMode is the pixel look's choice; a smooth look has one mode, so
	// read the mode to show the screen in through Mode.
	ScaleMode     ScaleMode `json:"scale_mode"`
	Fullscreen    bool      `json:"fullscreen"`
	WindowW       int       `json:"window_width"`
	WindowH       int       `json:"window_height"`
	SoundOn       bool      `json:"sound_on"`
	MusicOn       bool      `json:"music_on"`
	SFXOn         bool      `json:"sfx_on"`
	MasterVolume  float64   `json:"master_volume"` // 0–1
	MusicVolume   float64   `json:"music_volume"`  // 0–1
	SFXVolume     float64   `json:"sfx_volume"`    // 0–1
	ReducedMotion bool      `json:"reduced_motion"`
}

// DefaultSettings are the settings for looks l with no saved file: the
// first look.
func DefaultSettings(l Looks) Settings {
	w, h := l.First.DefaultWindow()
	s := Settings{
		ScaleMode: l.scaled().ScaleModes[0],
		WindowW:   w, WindowH: h,
		SoundOn: true, MusicOn: true, SFXOn: true,
		MasterVolume: 0.8, MusicVolume: 0.8, SFXVolume: 0.8,
	}
	if l.Two() {
		s.Look = l.First.Name
	}
	return s
}

// LoadSettings reads the settings at path for looks l. A missing or corrupt
// file gives the defaults, a missing field keeps its default, and values
// out of range are clamped, so it never fails.
func LoadSettings(path string, l Looks) Settings {
	s := DefaultSettings(l)
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	got := s
	if json.Unmarshal(data, &got) != nil {
		return s
	}
	return got.Clamped(l)
}

// Clamped returns s with every value made valid for looks l. An unknown
// look is the first.
func (s Settings) Clamped(l Looks) Settings {
	if l.Two() {
		s.Look = l.Named(s.Look).Name
	} else {
		s.Look = ""
	}
	if p := l.scaled(); !p.HasMode(s.ScaleMode) {
		s.ScaleMode = p.ScaleModes[0]
	}
	minW, minH := s.Shown(l).MinWindow()
	s.WindowW = min(max(s.WindowW, minW), MaxWindowW)
	s.WindowH = min(max(s.WindowH, minH), MaxWindowH)
	s.MasterVolume = unit(s.MasterVolume)
	s.MusicVolume = unit(s.MusicVolume)
	s.SFXVolume = unit(s.SFXVolume)
	return s
}

// Shown is the look the settings show, of looks l.
func (s Settings) Shown(l Looks) Preset { return l.Named(s.Look) }

// Mode is the scale mode to show the screen in: the saved one where the
// look shown offers it, otherwise that look's own.
func (s Settings) Mode(l Looks) ScaleMode {
	if p := s.Shown(l); !p.HasMode(s.ScaleMode) {
		return p.ScaleModes[0]
	}
	return s.ScaleMode
}

// AudioLevels are the master, music and effects volumes to play at: a
// switched-off part is silent. Sound off (Ctrl+S) silences everything.
func (s Settings) AudioLevels() (master, music, sfx float64) {
	if !s.SoundOn {
		return 0, 0, 0
	}
	music, sfx = s.MusicVolume, s.SFXVolume
	if !s.MusicOn {
		music = 0
	}
	if !s.SFXOn {
		sfx = 0
	}
	return s.MasterVolume, music, sfx
}

// unit clamps v to 0–1, treating NaN as 0.
func unit(v float64) float64 {
	if !(v >= 0) {
		return 0
	}
	return min(v, 1)
}

// SaveSettings writes the settings to path.
func SaveSettings(path string, s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return WriteFile(path, append(data, '\n'))
}

// WriteFile writes data to a temporary file beside path and renames it into
// place, so a crash mid-write cannot leave a half-written file.
func WriteFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp.Name(), path)
	}
	if werr != nil {
		os.Remove(tmp.Name())
	}
	return werr
}

// SaveDir is the folder the saved files live in: the folder holding the
// game's executable. Under go run the executable is built in a throwaway
// go-build folder, so the working folder is used instead.
func SaveDir() string {
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		dir := filepath.Dir(exe)
		if !inGoBuild(dir) {
			return dir
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// inGoBuild reports whether dir is inside a go-build temporary folder.
func inGoBuild(dir string) bool {
	for _, part := range strings.Split(filepath.ToSlash(dir), "/") {
		if strings.HasPrefix(part, "go-build") {
			return true
		}
	}
	return false
}
