package gfx

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Both kinds of font are Fonts.
var (
	_ Font = (*BitmapFont)(nil)
	_ Font = (*VectorFont)(nil)
)

func TestSmoothFonts(t *testing.T) {
	f := SmoothFonts(720)
	if f.Small.CellWidth() != 12 || f.Small.LineHeight() != 24 {
		t.Errorf("small font at 720: cells %d×%d, want 12×24", f.Small.CellWidth(), f.Small.LineHeight())
	}
	if f.Large.CellWidth() != 24 {
		t.Errorf("large font at 720: cells %d wide, want 24", f.Large.CellWidth())
	}
}

// TestVectorFontCells checks the cells against Go Mono's metrics: a
// character advances 0.6 em, and a line is 1.156 em tall. Each character is
// drawn at the start of its own cell, so it may be up to half a pixel wider
// or narrower than the cell, never more.
func TestVectorFontCells(t *testing.T) {
	for _, size := range []int{12, 16, 20, 25, 40} {
		f := NewVectorFont(size)
		if adv := text.Advance("M", f.face); math.Abs(adv-float64(f.CellWidth())) > 0.5 {
			t.Errorf("size %d: cell %d for an advance of %.3f", size, f.CellWidth(), adv)
		}
		if want := 0.6 * float64(size); math.Abs(float64(f.CellWidth())-want) > 0.5 {
			t.Errorf("size %d: CellWidth %d, want about %.1f", size, f.CellWidth(), want)
		}
		if want := 1.156 * float64(size); math.Abs(float64(f.LineHeight())-want) > 1 {
			t.Errorf("size %d: LineHeight %d, want about %.1f", size, f.LineHeight(), want)
		}
		if got := f.Width("ab é"); got != 4*f.CellWidth() {
			t.Errorf("size %d: Width of 4 characters %d, want %d", size, got, 4*f.CellWidth())
		}
	}
}
