package game

import (
	"image"
	"image/color"
	"testing"

	"github.com/comprofix/gameskeleton/internal/config"
)

func TestTestPatternShowsEveryColour(t *testing.T) {
	for _, p := range []config.Preset{config.Modern, config.Classic} {
		t.Run(p.Name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, p.ScreenWidth, p.ScreenHeight))
			drawTestPattern(img, p, 0)

			seen := map[color.RGBA]bool{}
			for i := 0; i < len(img.Pix); i += 4 {
				seen[color.RGBA{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}] = true
			}
			for i, c := range p.Palette.Colors {
				if !seen[c] {
					t.Errorf("colour %d (%s) is missing", i, p.Palette.Names[i])
				}
			}
			if len(seen) != p.Palette.Len() {
				t.Errorf("%d distinct colours drawn, want only the palette's %d", len(seen), p.Palette.Len())
			}

			fg := p.Palette.Colors[p.Palette.Brightest()]
			for _, pt := range []image.Point{{0, 0}, {p.ScreenWidth - 1, 0}, {0, p.ScreenHeight - 1}, {p.ScreenWidth - 1, p.ScreenHeight - 1}} {
				if got := img.RGBAAt(pt.X, pt.Y); got != fg {
					t.Errorf("frame corner %v = %v, want %v", pt, got, fg)
				}
			}
		})
	}
}

func TestTestPatternMarkerMoves(t *testing.T) {
	p := config.Modern
	a := image.NewRGBA(image.Rect(0, 0, p.ScreenWidth, p.ScreenHeight))
	b := image.NewRGBA(a.Bounds())
	drawTestPattern(a, p, 10)
	drawTestPattern(b, p, 11)
	y := p.ScreenHeight - patternMargin - 1
	fg := p.Palette.Colors[p.Palette.Brightest()]
	// The marker's left edge moves from x=18 to x=19.
	if a.RGBAAt(patternMargin+10, y) != fg || b.RGBAAt(patternMargin+10, y) == fg || b.RGBAAt(patternMargin+11, y) != fg {
		t.Error("the marker should move one pixel per tick")
	}
}
