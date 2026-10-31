// Package scenetest runs scenes headless for tests: no window,
// no sound, scripted input and a fixed seed, so a run is repeatable. It
// also compares results with golden files.
//
//	h := scenetest.New(t, 42)
//	h.Start(play.New(h.Ctx, config.Skills[0]))
//	h.Run(scenetest.Hold(30, ebiten.KeyArrowRight))
//	scenetest.Golden(t, "run", summary(h))
//
// Run the tests with -update to rewrite the golden files after an intended
// change.
package scenetest

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/input"
	"github.com/comprofix/gameskeleton/internal/scene"
	"github.com/comprofix/gameskeleton/internal/score"
)

var update = flag.Bool("update", false, "rewrite golden files with the current results")

// Harness runs a scene manager tick by tick from a script of inputs.
type Harness struct {
	T      testing.TB
	Ctx    *scene.Context
	Script *input.Script
	Mgr    *scene.Manager
}

// New returns a harness with a fresh context: the given seed, scripted
// input, the game's preset and default settings, empty high scores and no
// sound. Set more of Ctx (e.g. NewPlay or Audio) before calling Start.
func New(t testing.TB, seed uint64) *Harness {
	set := config.DefaultSettings(config.ProjectLooks())
	script := &input.Script{}
	return &Harness{
		T:      t,
		Script: script,
		Ctx: &scene.Context{
			Input:    input.New(script),
			RNG:      scene.NewRNG(seed),
			Preset:   config.Project,
			Looks:    config.ProjectLooks(),
			Settings: &set,
			Scores:   score.New(),
		},
	}
}

// Start makes s the running scene. The manager fades it in, which takes
// scene.FadeTicks.
func (h *Harness) Start(s scene.Scene) { h.Mgr = scene.NewManager(s) }

// Ticks runs n ticks.
func (h *Harness) Ticks(n int) {
	h.T.Helper()
	for range n {
		h.Ctx.Input.Update()
		if err := h.Mgr.Update(h.Ctx); err != nil {
			h.T.Fatal(err)
		}
		h.Ctx.Tick++
	}
}

// Run plays the steps, one tick each for their Ticks, and stops when they
// are done.
func (h *Harness) Run(steps ...input.Step) {
	h.T.Helper()
	h.Script.Steps = append(h.Script.Steps, steps...)
	for !h.Script.Done() {
		h.Ticks(1)
	}
}

// Press presses keys for a tick, lets go, and waits out any fade it caused.
func (h *Harness) Press(keys ...ebiten.Key) {
	h.T.Helper()
	h.Run(input.Press(keys...), input.Idle(1))
	h.Ticks(2 * scene.FadeTicks)
}

// Hold is a step that holds keys for n ticks.
func Hold(n int, keys ...ebiten.Key) input.Step { return input.Step{Keys: keys, Ticks: n} }

// Current is the running scene as a T, failing the test if it is not one.
func Current[T scene.Scene](h *Harness) T {
	h.T.Helper()
	s, ok := h.Mgr.Current().(T)
	if !ok {
		var want T
		h.T.Fatalf("current scene is %T, want %T", h.Mgr.Current(), want)
	}
	return s
}

// Golden compares got with testdata/name.golden, or writes it there when
// the tests run with -update.
func Golden(t testing.TB, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update to create it)", err)
	}
	if got != string(want) {
		t.Errorf("%s differs from the golden file.\ngot:\n%s\nwant:\n%s\n(run with -update if the change is intended)", path, got, want)
	}
}
