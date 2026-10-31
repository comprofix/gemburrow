// Package bitfont parses and draws bitmap fonts without a GPU,
// so both the game and the build tools can use them.
//
// The text format is one glyph after another: "@" plus the character on its
// own line ("@space" for space), then N rows of N characters, '#' for ink
// and '.' for empty. N is the font size, taken from the first row. Lines
// starting with ';' are comments.
package bitfont

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"io"
	"slices"
	"strings"
	"unicode/utf8"
)

// MaxSize is the largest supported glyph size.
const MaxSize = 32

// Font is a set of square glyphs. Each glyph is Size rows; bit Size-1 of a
// row is its leftmost pixel.
type Font struct {
	Size   int
	glyphs map[rune][]uint32
}

// Parse reads a font in the text format.
func Parse(r io.Reader) (*Font, error) {
	f := &Font{glyphs: map[rune][]uint32{}}
	sc := bufio.NewScanner(r)
	lineNo := 0
	var cur rune
	var rows []uint32
	inGlyph := false
	for sc.Scan() {
		lineNo++
		line := strings.TrimRight(sc.Text(), " \t\r")
		if strings.HasPrefix(line, ";") {
			continue
		}
		if !inGlyph {
			switch {
			case line == "":
				continue
			case line == "@space":
				cur = ' '
			case strings.HasPrefix(line, "@") && utf8.RuneCountInString(line) == 2:
				cur, _ = utf8.DecodeRuneInString(line[1:])
			default:
				return nil, fmt.Errorf("line %d: want a glyph header like \"@A\" or \"@space\", got %q", lineNo, line)
			}
			if _, dup := f.glyphs[cur]; dup {
				return nil, fmt.Errorf("line %d: duplicate glyph %q", lineNo, cur)
			}
			rows, inGlyph = nil, true
			continue
		}
		if f.Size == 0 {
			if len(line) == 0 || len(line) > MaxSize {
				return nil, fmt.Errorf("line %d: glyph rows must be 1–%d wide, got %d", lineNo, MaxSize, len(line))
			}
			f.Size = len(line)
		}
		if len(line) != f.Size {
			return nil, fmt.Errorf("line %d: glyph %q row is %d wide, want %d", lineNo, cur, len(line), f.Size)
		}
		var bits uint32
		for x, ch := range line {
			switch ch {
			case '#':
				bits |= 1 << (f.Size - 1 - x)
			case '.':
			default:
				return nil, fmt.Errorf("line %d: glyph %q has bad pixel %q", lineNo, cur, ch)
			}
		}
		rows = append(rows, bits)
		if len(rows) == f.Size {
			f.glyphs[cur] = rows
			inGlyph = false
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if inGlyph {
		return nil, fmt.Errorf("glyph %q is incomplete", cur)
	}
	if len(f.glyphs) == 0 {
		return nil, fmt.Errorf("no glyphs")
	}
	return f, nil
}

// Runes returns the font's characters in order.
func (f *Font) Runes() []rune {
	rs := make([]rune, 0, len(f.glyphs))
	for r := range f.glyphs {
		rs = append(rs, r)
	}
	slices.Sort(rs)
	return rs
}

// Has reports whether the font has a glyph for r.
func (f *Font) Has(r rune) bool {
	_, ok := f.glyphs[r]
	return ok
}

// Ink reports whether pixel (x, y) of r's glyph is ink. Pixels outside the
// glyph, and characters the font lacks, are empty.
func (f *Font) Ink(r rune, x, y int) bool {
	g, ok := f.glyphs[r]
	if !ok || x < 0 || y < 0 || x >= f.Size || y >= f.Size {
		return false
	}
	return g[y]&(1<<(f.Size-1-x)) != 0
}

// Width is the width of s in pixels: one glyph cell per character.
func (f *Font) Width(s string) int { return utf8.RuneCountInString(s) * f.Size }

// Draw writes s into dst at (x, y) in colour c. Characters the font lacks
// are drawn as '?'.
func (f *Font) Draw(dst *image.RGBA, s string, x, y int, c color.RGBA) {
	for _, r := range s {
		if !f.Has(r) {
			r = '?'
		}
		for gy := range f.Size {
			for gx := range f.Size {
				if f.Ink(r, gx, gy) && image.Pt(x+gx, y+gy).In(dst.Bounds()) {
					dst.SetRGBA(x+gx, y+gy, c)
				}
			}
		}
		x += f.Size
	}
}

// Scale2x returns a font twice the size, enlarged with the Scale2x (EPX)
// algorithm, which rounds off diagonal steps instead of just doubling
// pixels.
func Scale2x(f *Font) *Font {
	n := f.Size * 2
	out := &Font{Size: n, glyphs: make(map[rune][]uint32, len(f.glyphs))}
	for r := range f.glyphs {
		rows := make([]uint32, n)
		set := func(x, y int) { rows[y] |= 1 << (n - 1 - x) }
		for y := range f.Size {
			for x := range f.Size {
				p := f.Ink(r, x, y)
				a := f.Ink(r, x, y-1) // above
				b := f.Ink(r, x+1, y) // right
				c := f.Ink(r, x-1, y) // left
				d := f.Ink(r, x, y+1) // below
				e0, e1, e2, e3 := p, p, p, p
				if c == a && c != d && a != b {
					e0 = a
				}
				if a == b && a != c && b != d {
					e1 = b
				}
				if d == c && d != b && c != a {
					e2 = c
				}
				if b == d && b != a && d != c {
					e3 = d
				}
				if e0 {
					set(2*x, 2*y)
				}
				if e1 {
					set(2*x+1, 2*y)
				}
				if e2 {
					set(2*x, 2*y+1)
				}
				if e3 {
					set(2*x+1, 2*y+1)
				}
			}
		}
		out.glyphs[r] = rows
	}
	return out
}
