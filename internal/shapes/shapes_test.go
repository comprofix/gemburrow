package shapes

import (
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/gfx"
	"github.com/comprofix/gameskeleton/internal/input"
)

func TestContains(t *testing.T) {
	r := R(10, 20, 30, 40)
	in := []image.Point{{10, 20}, {39, 59}, {25, 40}}
	out := []image.Point{{9, 20}, {10, 19}, {40, 30}, {20, 60}}
	for _, p := range in {
		if !r.Contains(p) {
			t.Errorf("%v should be inside %v", p, r)
		}
	}
	for _, p := range out {
		if r.Contains(p) {
			t.Errorf("%v should be outside %v", p, r)
		}
	}
	// Side by side, no pixel is in both.
	a, b := R(0, 0, 10, 10), R(10, 0, 10, 10)
	for x := range 20 {
		if p := image.Pt(x, 5); a.Contains(p) == b.Contains(p) {
			t.Errorf("x %d is in both or neither", x)
		}
	}
}

func TestInsetOffsetCentred(t *testing.T) {
	r := R(10, 20, 30, 40)
	if got := r.Inset(5); got != R(15, 25, 20, 30) {
		t.Errorf("inset 5: %v", got)
	}
	if got := r.Inset(-2); got != R(8, 18, 34, 44) {
		t.Errorf("inset -2: %v", got)
	}
	if got := r.Inset(100); got != R(25, 40, 0, 0) {
		t.Errorf("inset past nothing: %v, want empty at the middle", got)
	}
	if got := r.Offset(3, -4); got != R(13, 16, 30, 40) {
		t.Errorf("offset: %v", got)
	}
	if x, y := r.Centre(); x != 25 || y != 40 {
		t.Errorf("centre %v, %v", x, y)
	}
	if got := r.Centred(10, 8); got != R(20, 36, 10, 8) {
		t.Errorf("centred: %v", got)
	}
}

func TestRadius(t *testing.T) {
	r := R(0, 0, 100, 20)
	for _, c := range []struct{ in, want float64 }{{5, 5}, {10, 10}, {30, 10}, {-3, 0}} {
		if got := r.Radius(c.in); got != c.want {
			t.Errorf("radius %v: %v, want %v", c.in, got, c.want)
		}
	}
}

func TestArc(t *testing.T) {
	pts := Arc(10, 20, 5, 0, math.Pi/2, 4)
	if len(pts) != 10 {
		t.Fatalf("%d numbers, want 5 points", len(pts))
	}
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	// From the right of the centre, clockwise to below it (y grows down).
	if !near(pts[0], 15) || !near(pts[1], 20) || !near(pts[8], 10) || !near(pts[9], 25) {
		t.Errorf("ends %v, %v", pts[:2], pts[8:])
	}
	for i := 0; i < len(pts); i += 2 {
		if d := math.Hypot(pts[i]-10, pts[i+1]-20); !near(d, 5) {
			t.Errorf("point %d is %v from the centre", i/2, d)
		}
	}
}

func TestStyles(t *testing.T) {
	sc := gfx.Scheme{Backdrop: 1, Panel: 2, Border: 3, Text: 4, Dim: 5, Highlight: 6, Title: 7, Shadow: 8}
	p := PanelFor(sc, 24)
	if p.Fill != 2 || p.Border != 3 || p.Shadow != 8 || p.Light != None || p.Radius != 8 || p.BorderWidth != 2 || p.Depth != 4 {
		t.Errorf("panel %+v", p)
	}
	if small := PanelFor(sc, 8); small.BorderWidth != 1 {
		t.Errorf("border %v at size 8, want at least 1", small.BorderWidth)
	}
	b := ButtonFor(sc, 24)
	if b.Text != [...]int{4, 6, 8, 5} {
		t.Errorf("label colours %v", b.Text)
	}
	if b.Panel[Normal] != p || b.Panel[Hover].Border != 6 || b.Panel[Pressed].Fill != 6 ||
		b.Panel[Disabled].Border != 5 || b.Panel[Disabled].Shadow != None {
		t.Errorf("button panels %+v", b.Panel)
	}
}

func TestButtonByMouse(t *testing.T) {
	b := Button{Rect: R(100, 50, 80, 30), Label: "OK"}
	held := input.Step{Pointer: &image.Point{120, 60}, Mouse: []ebiten.MouseButton{ebiten.MouseButtonLeft}, Ticks: 1}
	in := input.New(&input.Script{Steps: []input.Step{
		input.MoveTo(10, 10),  // away
		input.MoveTo(110, 55), // over
		input.Click(120, 60),  // pressed and clicked
		held,                  // still pressed, but not a new click
		input.MoveTo(300, 60), // gone
		input.Click(300, 60),  // a click elsewhere
	}})
	want := []struct {
		st      State
		clicked bool
	}{{Normal, false}, {Hover, false}, {Pressed, true}, {Pressed, false}, {Normal, false}, {Normal, false}}
	for i, w := range want {
		in.Update()
		if st, c := b.State(in), b.Clicked(in); st != w.st || c != w.clicked {
			t.Errorf("tick %d: state %v, clicked %v; want %v, %v", i, st, c, w.st, w.clicked)
		}
	}
	off := Button{Rect: b.Rect, Disabled: true}
	in = input.New(&input.Script{Steps: []input.Step{input.Click(120, 60)}})
	in.Update()
	if off.State(in) != Disabled || off.Clicked(in) {
		t.Error("a disabled button took a click")
	}
}
