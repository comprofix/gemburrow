package game

import (
	"image"
	"image/color"

	"github.com/comprofix/gameskeleton/internal/config"
)

// The test pattern's layout, in screen pixels.
const (
	patternMargin   = 8
	patternMaxCols  = 16 // swatches per row
	patternMarkSize = 8  // the moving marker
)

// drawTestPattern fills dst with a pattern for checking the display with
// any preset: every palette colour as a swatch, a one-pixel
// frame and checkerboard to show that scaling keeps pixels crisp, a diagonal
// line, and a marker that moves one pixel per tick to show smooth motion.
// It only writes pixels, so it runs in tests without a window.
func drawTestPattern(dst *image.RGBA, p config.Preset, tick int64) {
	pal := p.Palette
	w, h := p.ScreenWidth, p.ScreenHeight
	bg := pal.Colors[pal.Darkest()]
	fg := pal.Colors[pal.Brightest()]

	fillRect(dst, 0, 0, w, h, bg)

	// One-pixel frame around the whole screen.
	fillRect(dst, 0, 0, w, 1, fg)
	fillRect(dst, 0, h-1, w, 1, fg)
	fillRect(dst, 0, 0, 1, h, fg)
	fillRect(dst, w-1, 0, 1, h, fg)

	// Palette swatches across the top half, each outlined so the swatch in
	// the background colour still shows.
	n := pal.Len()
	cols := min(n, patternMaxCols)
	rows := (n + cols - 1) / cols
	sw := (w - 2*patternMargin) / cols
	sh := min(sw, (h/2-patternMargin)/rows)
	for i, c := range pal.Colors {
		x := patternMargin + (i%cols)*sw
		y := patternMargin + (i/cols)*sh
		fillRect(dst, x+1, y+1, sw-2, sh-2, fg)
		fillRect(dst, x+2, y+2, sw-4, sh-4, c)
	}

	// One-pixel checkerboard in the lower left, clear of the marker's path.
	top := h/2 + patternMargin
	cb := min(w/4, h/2-3*patternMargin-patternMarkSize)
	for y := 0; y < cb; y++ {
		for x := 0; x < cb; x++ {
			if (x+y)%2 == 0 {
				dst.SetRGBA(patternMargin+x, top+y, fg)
			}
		}
	}

	// A diagonal line beside it.
	for i := 0; i < cb; i++ {
		dst.SetRGBA(2*patternMargin+cb+i, top+i, fg)
	}

	// The marker sweeps across the bottom, one pixel per tick.
	span := w - 2*patternMargin - patternMarkSize
	x := patternMargin + int(tick%int64(span))
	y := h - patternMargin - patternMarkSize
	fillRect(dst, x, y, patternMarkSize, patternMarkSize, fg)
}

// fillRect fills a rectangle of dst with c, clipped to dst.
func fillRect(dst *image.RGBA, x, y, w, h int, c color.RGBA) {
	r := image.Rect(x, y, x+w, y+h).Intersect(dst.Bounds())
	for py := r.Min.Y; py < r.Max.Y; py++ {
		for px := r.Min.X; px < r.Max.X; px++ {
			dst.SetRGBA(px, py, c)
		}
	}
}
