package gfx

import (
	"image"
	"testing"
)

func TestAnimator(t *testing.T) {
	a := Animator{TicksPerFrame: 3, Frames: 4}
	got := []int{}
	for range 13 {
		got = append(got, a.Frame())
		a.Advance(1)
	}
	exp := []int{0, 0, 0, 1, 1, 1, 2, 2, 2, 3, 3, 3, 0}
	for i := range exp {
		if got[i] != exp[i] {
			t.Fatalf("frames %v, want %v", got, exp)
		}
	}
	a.Reset()
	a.Advance(7) // several frames in one step
	if a.Frame() != 2 {
		t.Errorf("after 7 ticks frame %d, want 2", a.Frame())
	}
	var zero Animator
	zero.Advance(10) // must not loop forever or divide by zero
	if zero.Frame() != 0 {
		t.Error("a zero Animator should stay on frame 0")
	}
}

func TestParseIndex(t *testing.T) {
	sheet := image.Rect(0, 0, 32, 16)
	idx, err := parseIndex([]byte(`{"a":{"x":0,"y":0,"w":16,"h":16,"frames":2}}`), sheet)
	if err != nil || idx["a"].Frames != 2 {
		t.Fatalf("got %+v, %v", idx, err)
	}
	bad := []string{
		`{"a":{"x":0,"y":0,"w":16,"h":16,"frames":3}}`, // runs off the right
		`{"a":{"x":0,"y":8,"w":16,"h":16,"frames":1}}`, // runs off the bottom
		`{"a":{"x":0,"y":0,"w":0,"h":16,"frames":1}}`,  // zero width
		`{"a":{"x":0,"y":0,"w":16,"h":16,"frames":0}}`, // no frames
		`not json`,
	}
	for _, b := range bad {
		if _, err := parseIndex([]byte(b), sheet); err == nil {
			t.Errorf("%s: no error", b)
		}
	}
}
