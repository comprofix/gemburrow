package input

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/gfx"
)

// run plays steps through an Input and returns it after each tick.
func run(steps []Step, bindings ...Binding) (*Input, *Script) {
	s := &Script{Steps: steps}
	return New(s, bindings...), s
}

func TestKeyboardActions(t *testing.T) {
	cases := []struct {
		keys []ebiten.Key
		want Action
	}{
		{[]ebiten.Key{ebiten.KeyArrowUp}, Up},
		{[]ebiten.Key{ebiten.KeyW}, Up},
		{[]ebiten.Key{ebiten.KeyS}, Down},
		{[]ebiten.Key{ebiten.KeyA}, Left},
		{[]ebiten.Key{ebiten.KeyArrowRight}, Right},
		{[]ebiten.Key{ebiten.KeyEnter}, Confirm},
		{[]ebiten.Key{ebiten.KeySpace}, Confirm},
		{[]ebiten.Key{ebiten.KeyEscape}, Back},
		{[]ebiten.Key{ebiten.KeyEscape}, Pause},
		{[]ebiten.Key{ebiten.KeyP}, Pause},
		{[]ebiten.Key{ebiten.KeyControl, ebiten.KeyS}, Sound},
		{[]ebiten.Key{ebiten.KeyControlLeft, ebiten.KeyR}, Restart},
		{[]ebiten.Key{ebiten.KeyControl, ebiten.KeyM}, Menu},
		{[]ebiten.Key{ebiten.KeyF9}, ScaleMode},
		{[]ebiten.Key{ebiten.KeyF10}, Quit},
		{[]ebiten.Key{ebiten.KeyF11}, Fullscreen},
		{[]ebiten.Key{ebiten.KeyAlt, ebiten.KeyEnter}, Fullscreen},
	}
	for _, c := range cases {
		in, _ := run([]Step{{Keys: c.keys, Ticks: 1}})
		in.Update()
		if !in.JustPressed(c.want) {
			t.Errorf("%v should press %v", c.keys, c.want)
		}
	}
}

func TestGamepadActions(t *testing.T) {
	cases := []struct {
		step Step
		want Action
	}{
		{Step{Buttons: []Button{PadA}}, Confirm},
		{Step{Buttons: []Button{PadB}}, Back},
		{Step{Buttons: []Button{PadStart}}, Pause},
		{Step{Buttons: []Button{PadSelect}}, Menu},
		{Step{Buttons: []Button{PadUp}}, Up},
		{Step{Buttons: []Button{PadRight}}, Right},
		{Step{StickY: -0.9}, Up},
		{Step{StickX: 0.9}, Right},
	}
	for _, c := range cases {
		in, _ := run([]Step{c.step})
		in.Update()
		if !in.JustPressed(c.want) {
			t.Errorf("%+v should press %v", c.step, c.want)
		}
	}
	in, _ := run([]Step{{StickX: 0.3, StickY: -0.3}})
	in.Update()
	if in.Held(Right) || in.Held(Up) {
		t.Error("a small stick push inside the dead zone should do nothing")
	}
}

func TestCtrlLettersAreNotMovement(t *testing.T) {
	in, _ := run([]Step{{Keys: []ebiten.Key{ebiten.KeyControl, ebiten.KeyS}}})
	in.Update()
	if in.Held(Down) {
		t.Error("Ctrl+S must not move down")
	}
	in, _ = run([]Step{{Keys: []ebiten.Key{ebiten.KeyControl, ebiten.KeyP}}})
	in.Update()
	if in.Held(Pause) {
		t.Error("Ctrl+P must not pause")
	}
}

func TestAltEnterIsNotConfirm(t *testing.T) {
	// Alt+Enter, then Alt released while Enter is still held: Confirm must
	// not fire until Enter is pressed again.
	in, _ := run([]Step{
		{Keys: []ebiten.Key{ebiten.KeyAlt, ebiten.KeyEnter}, Ticks: 1},
		{Keys: []ebiten.Key{ebiten.KeyEnter}, Ticks: 1},
		Idle(1),
		Press(ebiten.KeyEnter),
	})
	for i, want := range []bool{false, false, false, true} {
		in.Update()
		if in.JustPressed(Confirm) != want {
			t.Errorf("tick %d: Confirm = %v, want %v", i, !want, want)
		}
	}
}

func TestPressHoldRelease(t *testing.T) {
	in, _ := run([]Step{{Keys: []ebiten.Key{ebiten.KeySpace}, Ticks: 3}, Idle(1)})
	in.Update()
	if !in.JustPressed(Confirm) || !in.Held(Confirm) {
		t.Error("tick 0: want just pressed and held")
	}
	in.Update()
	if in.JustPressed(Confirm) || !in.Held(Confirm) {
		t.Error("tick 1: want held only")
	}
	in.Update()
	in.Update()
	if !in.JustReleased(Confirm) || in.Held(Confirm) {
		t.Error("tick 3: want just released")
	}
}

func TestRepeat(t *testing.T) {
	in, _ := run([]Step{{Keys: []ebiten.Key{ebiten.KeyArrowDown}, Ticks: RepeatDelay + 2*RepeatEvery + 1}})
	var fired []int
	for tick := range RepeatDelay + 2*RepeatEvery + 1 {
		in.Update()
		if in.Repeat(Down) {
			fired = append(fired, tick)
		}
	}
	want := []int{0, RepeatDelay + RepeatEvery - 1, RepeatDelay + 2*RepeatEvery - 1}
	if len(fired) != len(want) {
		t.Fatalf("fired on ticks %v, want %v", fired, want)
	}
	for i := range want {
		if fired[i] != want[i] {
			t.Fatalf("fired on ticks %v, want %v", fired, want)
		}
	}
}

func TestAnyJustPressed(t *testing.T) {
	cases := []struct {
		name string
		step Step
		want bool
	}{
		{"letter", Press(ebiten.KeyQ), true},
		{"gamepad", Step{Buttons: []Button{PadA}}, true},
		{"shift alone", Press(ebiten.KeyShift), false},
		{"window key", Press(ebiten.KeyF11), false},
		{"ctrl command", Press(ebiten.KeyControl, ebiten.KeyS), false},
		{"alt+enter", Press(ebiten.KeyAlt, ebiten.KeyEnter), false},
		{"click", Click(5, 5), true},
		{"right click", Step{Mouse: []ebiten.MouseButton{ebiten.MouseButtonRight}}, true},
		{"pointer moved", MoveTo(5, 5), false},
	}
	for _, c := range cases {
		in, _ := run([]Step{c.step})
		in.Update()
		if got := in.AnyJustPressed(); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	// A key held from the previous tick is not a new press.
	in, _ := run([]Step{{Keys: []ebiten.Key{ebiten.KeyQ}, Ticks: 2}})
	in.Update()
	in.Update()
	if in.AnyJustPressed() {
		t.Error("a held key counted as a new press")
	}
}

func TestGameBindings(t *testing.T) {
	jump := NumActions
	in, _ := run([]Step{Press(ebiten.KeyX)}, Binding{Action: jump, Keys: []ebiten.Key{ebiten.KeyX}, Buttons: []Button{PadA}})
	in.Update()
	if !in.JustPressed(jump) {
		t.Error("the game's own action did not fire")
	}
}

func TestScriptDone(t *testing.T) {
	in, s := run([]Step{Idle(2)})
	in.Update()
	if s.Done() {
		t.Error("done too early")
	}
	in.Update()
	if !s.Done() {
		t.Error("not done after its steps")
	}
	in.Update() // reading past the end is harmless
}

// TestPointerThroughView: in every scale mode the pointer is turned into
// screen pixels through the rectangle the screen is shown in, and is not
// over the screen in the letterbox around it.
func TestPointerThroughView(t *testing.T) {
	const sw, sh = 640, 360
	for _, mode := range []config.ScaleMode{config.ScaleInteger, config.ScaleCRT, config.ScaleSmooth} {
		for _, win := range []image.Point{{1000, 800}, {1920, 1080}, {500, 700}} {
			r := gfx.Fit(mode, sw, sh, win.X, win.Y)
			in := &Input{}
			in.SetView(View{X: r.X, Y: r.Y, W: r.W, H: r.H, Width: sw, Height: sh})
			// at is the window position of the middle of screen pixel (x, y).
			at := func(x, y int) (float64, float64) {
				return r.X + (float64(x)+0.5)*r.W/sw, r.Y + (float64(y)+0.5)*r.H/sh
			}
			for _, want := range []image.Point{{0, 0}, {sw - 1, sh - 1}, {sw / 2, sh / 2}, {17, 300}} {
				wx, wy := at(want.X, want.Y)
				if got, ok := in.toScreen(wx, wy); got != want || !ok {
					t.Errorf("%s in %v: pixel %v read as %v, %v", mode, win, want, got, ok)
				}
			}
			outside := []image.Point{{-1, sh / 2}, {sw, sh / 2}, {sw / 2, -1}, {sw / 2, sh}}
			for _, o := range outside {
				wx, wy := at(o.X, o.Y)
				if _, ok := in.toScreen(wx, wy); ok {
					t.Errorf("%s in %v: %v is in the letterbox but read as on the screen", mode, win, o)
				}
			}
		}
	}
	// With no view set, window pixels are screen pixels.
	if p, ok := (&Input{}).toScreen(12.7, 3); p != image.Pt(12, 3) || !ok {
		t.Errorf("no view: %v, %v", p, ok)
	}
}

func TestMouse(t *testing.T) {
	in, _ := run([]Step{
		Click(10, 20), Idle(1), MoveTo(30, 40),
		{Wheel: -1, Ticks: 1},
		{Mouse: []ebiten.MouseButton{ebiten.MouseButtonRight}, Ticks: 1},
	})
	type tick struct {
		at                   image.Point
		moved, clicked, held bool
		wheel                float64
		back                 bool
	}
	want := []tick{
		{image.Pt(10, 20), true, true, true, 0, false},
		{image.Pt(10, 20), false, false, false, 0, false}, // the pointer stays put
		{image.Pt(30, 40), true, false, false, 0, false},
		{image.Pt(30, 40), false, false, false, -1, false},
		{image.Pt(30, 40), false, false, false, 0, true}, // a right click is Back and Pause
	}
	for i, w := range want {
		in.Update()
		p, _ := in.Pointer()
		got := tick{p, in.PointerMoved(), in.Clicked(), in.MouseHeld(), in.Wheel(), in.JustPressed(Back) && in.JustPressed(Pause)}
		if got != w {
			t.Errorf("tick %d: got %+v, want %+v", i, got, w)
		}
	}
}
