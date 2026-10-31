// Package gfx holds the palette access, scaling and drawing helpers.
package gfx

import (
	"math"

	"github.com/comprofix/gameskeleton/internal/config"
)

// Rect is a destination rectangle on the output surface, in output pixels.
type Rect struct {
	X, Y, W, H float64
}

// Fit returns where an sw×sh screen is drawn on an outW×outH output surface
// in the given mode, centred with letterboxing.
func Fit(mode config.ScaleMode, sw, sh, outW, outH int) Rect {
	ow, oh := float64(outW), float64(outH)
	fw, fh := float64(sw), float64(sh)
	var w, h float64
	switch mode {
	case config.ScaleCRT:
		// The largest 4:3 rectangle that fits.
		w = math.Min(ow, oh*4/3)
		h = w * 3 / 4
	case config.ScaleSmooth:
		s := math.Min(ow/fw, oh/fh)
		w, h = fw*s, fh*s
	default:
		s := math.Floor(math.Min(ow/fw, oh/fh))
		if s < 1 {
			// Window smaller than the screen: fall back to a fractional fit.
			s = math.Min(ow/fw, oh/fh)
		}
		w, h = fw*s, fh*s
	}
	return Rect{X: math.Floor((ow - w) / 2), Y: math.Floor((oh - h) / 2), W: w, H: h}
}

// PrescaleFactor is the whole-number nearest-neighbour upscale applied to
// an sw×sh screen before a smooth resample to the non-whole-number size r.
// Upscaling first keeps pixel edges sharp while evening out uneven rows.
func PrescaleFactor(r Rect, sw, sh int) int {
	k := math.Ceil(math.Max(r.W/float64(sw), r.H/float64(sh)))
	return int(math.Max(k, 1))
}
