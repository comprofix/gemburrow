package gfx

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/spritegrid"
)

// Sprite is a named sprite's frames within a sheet.
type Sprite struct {
	frames []*ebiten.Image
	W, H   int
}

// Frames returns the number of frames.
func (s *Sprite) Frames() int { return len(s.frames) }

// Frame returns frame i, wrapping around.
func (s *Sprite) Frame(i int) *ebiten.Image {
	n := len(s.frames)
	return s.frames[((i%n)+n)%n]
}

// Draw draws frame i with its top-left corner at (x, y). With flipX the
// frame is mirrored, so a right-facing sprite faces left; it still covers
// the same rectangle.
func (s *Sprite) Draw(dst *ebiten.Image, i int, x, y float64, flipX bool) {
	op := &ebiten.DrawImageOptions{}
	if flipX {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(float64(s.W), 0)
	}
	op.GeoM.Translate(x, y)
	dst.DrawImage(s.Frame(i), op)
}

// Sheet holds the sprites from one generated sheet.
type Sheet struct {
	sprites map[string]*Sprite
}

// LoadSheet reads base.png and base.json, as written by tools/spritegen.
func LoadSheet(fsys fs.FS, base string) (*Sheet, error) {
	pf, err := fsys.Open(base + ".png")
	if err != nil {
		return nil, err
	}
	defer pf.Close()
	img, err := png.Decode(pf)
	if err != nil {
		return nil, fmt.Errorf("%s.png: %w", base, err)
	}
	data, err := fs.ReadFile(fsys, base+".json")
	if err != nil {
		return nil, err
	}
	index, err := parseIndex(data, img.Bounds())
	if err != nil {
		return nil, fmt.Errorf("%s.json: %w", base, err)
	}

	atlas := ebiten.NewImageFromImage(img)
	sh := &Sheet{sprites: make(map[string]*Sprite, len(index))}
	for name, e := range index {
		sp := &Sprite{W: e.W, H: e.H}
		for f := range e.Frames {
			r := image.Rect(e.X+f*e.W, e.Y, e.X+(f+1)*e.W, e.Y+e.H)
			sp.frames = append(sp.frames, atlas.SubImage(r).(*ebiten.Image))
		}
		sh.sprites[name] = sp
	}
	return sh, nil
}

// parseIndex reads a sheet's frame index and checks every frame lies within
// the sheet image.
func parseIndex(data []byte, bounds image.Rectangle) (map[string]spritegrid.Entry, error) {
	var index map[string]spritegrid.Entry
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}
	for name, e := range index {
		r := image.Rect(e.X, e.Y, e.X+e.W*e.Frames, e.Y+e.H)
		if e.W <= 0 || e.H <= 0 || e.Frames <= 0 || !r.In(bounds) {
			return nil, fmt.Errorf("sprite %q (%+v) does not fit the %v sheet", name, e, bounds.Size())
		}
	}
	return index, nil
}

// Sprite returns the named sprite. It panics if the name is missing, since
// sprite names are fixed at build time.
func (sh *Sheet) Sprite(name string) *Sprite {
	sp, ok := sh.sprites[name]
	if !ok {
		panic("gfx: no sprite named " + name)
	}
	return sp
}
