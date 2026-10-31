package config

import (
	"github.com/comprofix/gameskeleton/internal/gfx/palettes"
	"github.com/comprofix/gameskeleton/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/gameskeleton/internal/gfx/palettes/sweetie16"
)

// ScaleMode is how the internal screen is shown in the window.
// The values are what settings.json stores.
type ScaleMode string

const (
	// ScaleInteger is the largest whole-number scale that fits, letterboxed.
	ScaleInteger ScaleMode = "integer"
	// ScaleCRT stretches the screen to a 4:3 shape, as on an old monitor.
	ScaleCRT ScaleMode = "crt"
	// ScaleSmooth is the largest size that fits, by any factor, smoothed
	// and letterboxed: for the smooth art style.
	ScaleSmooth ScaleMode = "smooth"
)

// ArtStyle is how a game's art is made, which decides its fonts and
// scaling.
type ArtStyle string

const (
	// ArtPixel is pixel art: sprites and bitmap fonts at a low resolution,
	// scaled up by whole numbers.
	ArtPixel ArtStyle = "pixel"
	// ArtSmooth is smooth art: shapes drawn in code and an outline font at
	// a high resolution, scaled by any factor.
	ArtSmooth ArtStyle = "smooth"
)

// Preset is a tested set of display choices a game starts from.
type Preset struct {
	Name                      string
	ScreenWidth, ScreenHeight int
	ScaleModes                []ScaleMode // the modes F9 cycles through; the first is the default
	Palette                   palettes.Palette
	Audio                     AudioStyle
	Art                       ArtStyle
}

// Modern is 640×360 with Resurrect 64, scaled by whole numbers, with
// four-channel chip sound.
var Modern = Preset{
	Name:        "Modern",
	ScreenWidth: 640, ScreenHeight: 360,
	ScaleModes: []ScaleMode{ScaleInteger},
	Palette:    resurrect64.Palette,
	Audio:      Chip4,
	Art:        ArtPixel,
}

// Classic is 320×200 with Sweetie 16, scaled by whole numbers or to CRT 4:3,
// with a single PC-speaker voice.
var Classic = Preset{
	Name:        "Classic PC",
	ScreenWidth: 320, ScreenHeight: 200,
	ScaleModes: []ScaleMode{ScaleInteger, ScaleCRT},
	Palette:    sweetie16.Palette,
	Audio:      Speaker,
	Art:        ArtPixel,
}

// Smooth is 1280×720 with Resurrect 64 and smooth art: shapes drawn in code
// and Go Mono Bold, scaled by any factor, with four-channel chip sound.
var Smooth = Preset{
	Name:        "Smooth",
	ScreenWidth: 1280, ScreenHeight: 720,
	ScaleModes: []ScaleMode{ScaleSmooth},
	Palette:    resurrect64.Palette,
	Audio:      Chip4,
	Art:        ArtSmooth,
}

// HasMode reports whether the preset offers mode m.
func (p Preset) HasMode(m ScaleMode) bool {
	for _, x := range p.ScaleModes {
		if x == m {
			return true
		}
	}
	return false
}

// NextMode is the mode after m in the preset's cycle, wrapping around.
func (p Preset) NextMode(m ScaleMode) ScaleMode {
	for i, x := range p.ScaleModes {
		if x == m {
			return p.ScaleModes[(i+1)%len(p.ScaleModes)]
		}
	}
	return p.ScaleModes[0]
}

// The window's allowed size, in window pixels.
const (
	MaxWindowW, MaxWindowH = 7680, 4320
	defaultWindowW         = 1280 // the default window is about this wide
)

// DefaultWindow is the largest whole-number multiple of the screen that is
// no wider than 1280 pixels: 1280×720 for Modern and Smooth, 1280×800 for
// Classic.
func (p Preset) DefaultWindow() (w, h int) {
	k := max(1, defaultWindowW/p.ScreenWidth)
	return p.ScreenWidth * k, p.ScreenHeight * k
}

// MinWindow is the smallest window: one window pixel per screen pixel for
// pixel art, and half the screen for smooth art, which scales by any
// factor.
func (p Preset) MinWindow() (w, h int) {
	if p.Art == ArtSmooth {
		return p.ScreenWidth / 2, p.ScreenHeight / 2
	}
	return p.ScreenWidth, p.ScreenHeight
}
