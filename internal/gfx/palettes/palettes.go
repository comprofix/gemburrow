// Package palettes defines the palette type shared by the ready-made
// palettes in its subpackages. Each subpackage names its colours
// as index constants; the palette files are the only place hex codes appear.
package palettes

import "image/color"

// Entry is one named colour, written as 0xRRGGBB.
type Entry struct {
	Name string
	Hex  uint32
}

// Palette is an ordered list of named colours. Colour i is Colors[i], named
// Names[i]; game code and sprites refer to colours by index or by name.
type Palette struct {
	Name   string
	Colors []color.RGBA
	Names  []string
}

// New builds a palette from its entries, in index order.
func New(name string, entries []Entry) Palette {
	p := Palette{
		Name:   name,
		Colors: make([]color.RGBA, len(entries)),
		Names:  make([]string, len(entries)),
	}
	for i, e := range entries {
		p.Colors[i] = color.RGBA{R: uint8(e.Hex >> 16), G: uint8(e.Hex >> 8), B: uint8(e.Hex), A: 0xff}
		p.Names[i] = e.Name
	}
	return p
}

// Len is the number of colours.
func (p Palette) Len() int { return len(p.Colors) }

// Lookup returns the index of the colour with the given name.
func (p Palette) Lookup(name string) (int, bool) {
	for i, n := range p.Names {
		if n == name {
			return i, true
		}
	}
	return 0, false
}

// Darkest and Brightest return the indices of the colours with the lowest
// and highest luminance, for backgrounds and highlights that must work with
// any palette.
func (p Palette) Darkest() int   { return p.extreme(func(a, b int) bool { return a < b }) }
func (p Palette) Brightest() int { return p.extreme(func(a, b int) bool { return a > b }) }

func (p Palette) extreme(better func(a, b int) bool) int {
	best, bestL := 0, luminance(p.Colors[0])
	for i, c := range p.Colors[1:] {
		if l := luminance(c); better(l, bestL) {
			best, bestL = i+1, l
		}
	}
	return best
}

// luminance is a cheap perceptual brightness (Rec. 601 weights, ×1000).
func luminance(c color.RGBA) int {
	return 299*int(c.R) + 587*int(c.G) + 114*int(c.B)
}
