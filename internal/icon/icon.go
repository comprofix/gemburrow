// Package icon builds the program icon from assets/sprites/icon.txt, at each
// size the window and the Windows exe need.
package icon

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/fs"

	"github.com/comprofix/gameskeleton/assets"
)

// sizes lists each icon size and the hand-drawn sprite it is enlarged
// from, always by a whole factor so the pixels stay sharp.
var sizes = []struct {
	size   int
	sprite string
}{
	{16, "icon16"},
	{32, "icon32"},
	{48, "icon16"},
	{64, "icon32"},
	{128, "icon32"},
	{256, "icon32"},
}

// Images returns the icon at every size, smallest first; the system picks
// the best one for each place it is shown. It reads the sheet with the
// standard library, since ebiten images can't be read back before the game
// starts.
func Images() ([]image.Image, error) {
	pf, err := assets.FS.Open("gen/icon.png")
	if err != nil {
		return nil, err
	}
	defer pf.Close()
	sheet, err := png.Decode(pf)
	if err != nil {
		return nil, fmt.Errorf("gen/icon.png: %w", err)
	}
	data, err := fs.ReadFile(assets.FS, "gen/icon.json")
	if err != nil {
		return nil, err
	}
	var index map[string]struct{ X, Y, W, H int }
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("gen/icon.json: %w", err)
	}

	var icons []image.Image
	for _, is := range sizes {
		e, ok := index[is.sprite]
		if !ok {
			return nil, fmt.Errorf("gen/icon.json: no sprite named %s", is.sprite)
		}
		icons = append(icons, enlarge(sheet, image.Rect(e.X, e.Y, e.X+e.W, e.Y+e.H), is.size))
	}
	return icons, nil
}

// enlarge copies the square r of src to a new size×size image, repeating
// each pixel size/r.Dx() times across and down.
func enlarge(src image.Image, r image.Rectangle, size int) *image.NRGBA {
	k := size / r.Dx()
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range r.Dy() {
		for x := range r.Dx() {
			c := src.At(r.Min.X+x, r.Min.Y+y)
			draw.Draw(dst, image.Rect(x*k, y*k, (x+1)*k, (y+1)*k), image.NewUniform(c), image.Point{}, draw.Src)
		}
	}
	return dst
}
