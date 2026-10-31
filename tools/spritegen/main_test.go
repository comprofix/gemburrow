package main

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/comprofix/gameskeleton/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/gameskeleton/internal/spritegrid"
)

func TestRun(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "sprites")
	os.MkdirAll(in, 0o755)
	os.WriteFile(filepath.Join(in, "test.txt"), []byte("colours k=Ink g=Gold\nsprite blob 2 2\nkg\ngk\n\ngk\nkg\n"), 0o644)

	o := options{
		in:         in,
		out:        filepath.Join(dir, "gen"),
		previewDir: filepath.Join(dir, "preview"),
		fontPath:   "../../assets/font/font8x8.txt",
		scale:      4,
		pal:        resurrect64.Palette,
	}
	if err := run(o); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(o.out, "test.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index map[string]spritegrid.Entry
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatal(err)
	}
	if index["blob"] != (spritegrid.Entry{W: 2, H: 2, Frames: 2}) {
		t.Errorf("index %+v", index)
	}

	f, err := os.Open(filepath.Join(o.out, "test.png"))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if r, g, b, _ := img.At(1, 0).RGBA(); r>>8 != 0xf9 || g>>8 != 0xc2 || b>>8 != 0x2b {
		t.Errorf("pixel (1,0) is not Gold")
	}

	pf, err := os.Open(filepath.Join(o.previewDir, "test.png"))
	if err != nil {
		t.Fatal("no preview written:", err)
	}
	pimg, err := png.Decode(pf)
	pf.Close()
	if err != nil {
		t.Fatal(err)
	}
	if pimg.Bounds().Dx()%4 != 0 || pimg.Bounds().Dx() < 4*len("blob 2x2 x2")*8 {
		t.Errorf("preview is %v; want 4× scale wide enough for the label", pimg.Bounds())
	}
}

func TestRunErrors(t *testing.T) {
	dir := t.TempDir()
	// A game with no sprites yet is fine: nothing is written.
	if err := run(options{in: dir, out: filepath.Join(dir, "gen"), pal: resurrect64.Palette}); err != nil {
		t.Errorf("an empty sprite folder: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "gen")); err == nil {
		t.Error("an empty sprite folder wrote output")
	}
	os.WriteFile(filepath.Join(dir, "bad.txt"), []byte("sprite a 1 1\nz\n"), 0o644)
	if err := run(options{in: dir, out: filepath.Join(dir, "gen"), pal: resurrect64.Palette}); err == nil {
		t.Error("a bad sprite file should be an error")
	}
}
