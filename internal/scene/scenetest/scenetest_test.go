package scenetest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/scene"
)

func TestHarnessRunsScenes(t *testing.T) {
	h := New(t, 1)
	h.Start(scene.NewTitle())
	h.Ticks(scene.FadeTicks)
	Current[*scene.Title](h)
	h.Press(ebiten.KeyQ)
	Current[*scene.MainMenu](h)
	if !h.Script.Done() || h.Ctx.Tick == 0 {
		t.Errorf("script done %v, tick %d", h.Script.Done(), h.Ctx.Tick)
	}
}

func TestSameSeedSameNumbers(t *testing.T) {
	a, b := New(t, 9), New(t, 9)
	for range 5 {
		if a.Ctx.RNG.Uint64() != b.Ctx.RNG.Uint64() {
			t.Fatal("the same seed gave different numbers")
		}
	}
}

// failRecorder is a testing.TB that records failures instead of failing.
type failRecorder struct {
	testing.TB
	failed bool
}

func (f *failRecorder) Helper()               {}
func (f *failRecorder) Errorf(string, ...any) { f.failed = true }
func (f *failRecorder) Fatalf(string, ...any) { f.failed = true }
func (f *failRecorder) Fatal(...any)          { f.failed = true }

func TestGolden(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	missing := &failRecorder{TB: t}
	Golden(missing, "x", "hello\n")
	if !missing.failed {
		t.Error("a missing golden file did not fail")
	}
	os.MkdirAll("testdata", 0o755)
	os.WriteFile(filepath.Join("testdata", "x.golden"), []byte("hello\n"), 0o644)
	same := &failRecorder{TB: t}
	Golden(same, "x", "hello\n")
	if same.failed {
		t.Error("a matching result failed")
	}
	differs := &failRecorder{TB: t}
	Golden(differs, "x", "goodbye\n")
	if !differs.failed {
		t.Error("a different result passed")
	}
}
