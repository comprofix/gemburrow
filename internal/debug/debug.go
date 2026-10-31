// Package debug holds what debug builds add. Build or run with
// -tags debug to turn it on:
//
//	go run -tags debug ./cmd/gameskeleton
//
// Enabled is a constant, so code behind "if debug.Enabled" is left out of
// normal builds altogether.
package debug

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/gameskeleton/internal/gfx"
)

// Box outlines a w×h rectangle at (x, y) in palette colour c, one pixel
// thick, e.g. to show a collision box.
func Box(dst *ebiten.Image, x, y, w, h float64, c int) {
	vector.StrokeRect(dst, float32(x)+0.5, float32(y)+0.5, float32(w)-1, float32(h)-1, 1, gfx.RGBA(c), false)
}
