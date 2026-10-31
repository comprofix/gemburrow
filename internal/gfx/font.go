package gfx

import (
	"fmt"
	"image"
	"image/color"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/bitfont"
)

// Font draws text in any palette colour. Every character is CellWidth
// pixels wide and each line LineHeight pixels tall, so text lays out in
// character cells.
type Font interface {
	// CellWidth is how far each character moves the next one on, in pixels.
	CellWidth() int
	// LineHeight is the height of one line of text, in pixels.
	LineHeight() int
	// Width is the width of s in pixels.
	Width(s string) int
	// Draw renders s with its top-left corner at (x, y) in palette colour c.
	Draw(dst *ebiten.Image, s string, x, y, c int)
	// DrawShadowed renders s with a drop shadow in palette colour shadow.
	DrawShadowed(dst *ebiten.Image, s string, x, y, c, shadow int)
	// CentreX is the x that centres s across dst.
	CentreX(dst *ebiten.Image, s string) int
}

// Fonts are the game's two fonts.
type Fonts struct {
	Small Font // 8×8
	Large Font // 16×16, made from Small with Scale2x
}

// BitmapFont is a Font drawn from a bitmap font's glyphs.
type BitmapFont struct {
	bf    *bitfont.Font
	atlas *ebiten.Image
	index map[rune]int
}

// LoadFonts reads the 8×8 font from fsys and makes the large font from it.
func LoadFonts(fsys fs.FS, path string) (Fonts, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return Fonts{}, err
	}
	defer f.Close()
	small, err := bitfont.Parse(f)
	if err != nil {
		return Fonts{}, fmt.Errorf("%s: %w", path, err)
	}
	return Fonts{Small: NewFont(small), Large: NewFont(bitfont.Scale2x(small))}, nil
}

// NewFont builds a glyph atlas for bf. Ink is stored white and tinted when
// drawn.
func NewFont(bf *bitfont.Font) *BitmapFont {
	runes := bf.Runes()
	n := bf.Size
	img := image.NewRGBA(image.Rect(0, 0, n*len(runes), n))
	f := &BitmapFont{bf: bf, index: make(map[rune]int, len(runes))}
	for i, r := range runes {
		bf.Draw(img.SubImage(image.Rect(i*n, 0, (i+1)*n, n)).(*image.RGBA), string(r), i*n, 0, color.RGBA{0xff, 0xff, 0xff, 0xff})
		f.index[r] = i
	}
	f.atlas = ebiten.NewImageFromImage(img)
	return f
}

// CellWidth is the font's cell size: 8 or 16, as its cells are square.
func (f *BitmapFont) CellWidth() int { return f.bf.Size }

// LineHeight is the font's cell size: 8 or 16, as its cells are square.
func (f *BitmapFont) LineHeight() int { return f.bf.Size }

// Width is the width of s in pixels.
func (f *BitmapFont) Width(s string) int { return f.bf.Width(s) }

// Draw renders s at (x, y) in palette colour c. Characters missing from the
// font are drawn as '?'.
func (f *BitmapFont) Draw(dst *ebiten.Image, s string, x, y, c int) {
	n := f.bf.Size
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleWithColor(RGBA(c))
	for _, r := range s {
		i, ok := f.index[r]
		if !ok {
			i, ok = f.index['?']
		}
		if ok && r != ' ' {
			src := f.atlas.SubImage(image.Rect(i*n, 0, (i+1)*n, n)).(*ebiten.Image)
			op.GeoM.Reset()
			op.GeoM.Translate(float64(x), float64(y))
			dst.DrawImage(src, op)
		}
		x += n
	}
}

// DrawShadowed renders s with a drop shadow in palette colour shadow, offset
// down and right by one pixel per 8 pixels of font size.
func (f *BitmapFont) DrawShadowed(dst *ebiten.Image, s string, x, y, c, shadow int) {
	d := max(1, f.bf.Size/8)
	f.Draw(dst, s, x+d, y+d, shadow)
	f.Draw(dst, s, x, y, c)
}

// CentreX is the x that centres s across dst.
func (f *BitmapFont) CentreX(dst *ebiten.Image, s string) int {
	return (dst.Bounds().Dx() - f.Width(s)) / 2
}
