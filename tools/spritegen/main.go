// Command spritegen converts text-grid sprite sources (*.txt) into PNG
// sprite sheets plus a JSON frame index, using the project palette.
// It runs through go generate in assets:
//
//	spritegen -in sprites -out gen
//
// With -preview DIR it also writes a scaled-up preview of each sheet, with
// sprite names on a checkerboard, for checking art by eye.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/comprofix/gameskeleton/internal/bitfont"
	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/gfx/palettes"
	"github.com/comprofix/gameskeleton/internal/spritegrid"
)

type options struct {
	in, out    string
	previewDir string
	fontPath   string
	scale      int
	pal        palettes.Palette
}

func main() {
	var o options
	flag.StringVar(&o.in, "in", "sprites", "directory of *.txt sprite sources")
	flag.StringVar(&o.out, "out", "gen", "output directory for *.png and *.json")
	flag.StringVar(&o.previewDir, "preview", "", "also write scaled-up previews to this directory")
	flag.StringVar(&o.fontPath, "font", "font/font8x8.txt", "font for preview labels")
	flag.IntVar(&o.scale, "scale", 4, "preview scale")
	flag.Parse()
	o.pal = config.Project.Palette
	if err := run(o); err != nil {
		fmt.Fprintln(os.Stderr, "spritegen:", err)
		os.Exit(1)
	}
}

func run(o options) error {
	srcs, err := filepath.Glob(filepath.Join(o.in, "*.txt"))
	if err != nil {
		return err
	}
	if len(srcs) == 0 {
		return nil // a game with no sprites yet: nothing to make
	}
	var font *bitfont.Font
	if o.previewDir != "" {
		if font, err = loadFont(o.fontPath); err != nil {
			return err
		}
		if err := os.MkdirAll(o.previewDir, 0o755); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(o.out, 0o755); err != nil {
		return err
	}
	for _, src := range srcs {
		if err := convert(src, o, font); err != nil {
			return fmt.Errorf("%s: %w", src, err)
		}
	}
	return nil
}

func loadFont(path string) (*bitfont.Font, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	font, err := bitfont.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return font, nil
}

func convert(src string, o options, font *bitfont.Font) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	sprites, err := spritegrid.Parse(f, o.pal)
	f.Close()
	if err != nil {
		return err
	}
	img, index, err := spritegrid.Pack(sprites, o.pal)
	if err != nil {
		return err
	}

	name := strings.TrimSuffix(filepath.Base(src), ".txt")
	base := filepath.Join(o.out, name)
	if err := writePNG(base+".png", img); err != nil {
		return err
	}
	// encoding/json sorts map keys, so the index diffs cleanly.
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(base+".json", append(data, '\n'), 0o644); err != nil {
		return err
	}

	if font != nil {
		return writePNG(filepath.Join(o.previewDir, name+".png"), preview(sprites, o.pal, font, o.scale))
	}
	return nil
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Preview layout, in unscaled pixels.
const (
	previewPad = 6
	frameGap   = 2
)

// Preview colours: neutral greys that never appear in sprites' meaning, so
// transparency shows as a checkerboard.
var (
	checkDark  = color.RGBA{0x40, 0x40, 0x48, 0xff}
	checkLight = color.RGBA{0x58, 0x58, 0x60, 0xff}
	labelBack  = color.RGBA{0x18, 0x18, 0x1c, 0xff}
	labelInk   = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

// preview draws every sprite with its name, size and frame count above its
// frames, on a checkerboard, then scales the whole image up.
func preview(sprites []spritegrid.Sprite, pal palettes.Palette, font *bitfont.Font, scale int) *image.RGBA {
	labels := make([]string, len(sprites))
	w, h := 0, previewPad
	for i, s := range sprites {
		labels[i] = fmt.Sprintf("%s %dx%d x%d", s.Name, s.W, s.H, len(s.Frames))
		w = max(w, font.Width(labels[i]), len(s.Frames)*(s.W+frameGap))
		h += font.Size + frameGap + s.H + previewPad
	}
	w += 2 * previewPad

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			c := checkDark
			if (x/2+y/2)%2 == 0 {
				c = checkLight
			}
			img.SetRGBA(x, y, c)
		}
	}
	y := previewPad
	for i, s := range sprites {
		for ly := y - 1; ly < y+font.Size+1; ly++ {
			for lx := 0; lx < w; lx++ {
				img.SetRGBA(lx, ly, labelBack)
			}
		}
		font.Draw(img, labels[i], previewPad, y, labelInk)
		y += font.Size + frameGap
		for f, frame := range s.Frames {
			x0 := previewPad + f*(s.W+frameGap)
			for p, c := range frame {
				if c != spritegrid.Transparent {
					img.SetRGBA(x0+p%s.W, y+p/s.W, pal.Colors[c])
				}
			}
		}
		y += s.H + previewPad
	}
	return scaleUp(img, max(scale, 1))
}

// scaleUp enlarges img by a whole number with nearest-neighbour sampling.
func scaleUp(img *image.RGBA, k int) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*k, b.Dy()*k))
	for y := range out.Bounds().Dy() {
		for x := range out.Bounds().Dx() {
			out.SetRGBA(x, y, img.RGBAAt(x/k, y/k))
		}
	}
	return out
}
