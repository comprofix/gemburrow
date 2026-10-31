// Package input maps the keyboard, gamepads and mouse to game actions.
// Each action reports whether it was just pressed, is held, or was just
// released. The mouse pointer is also reported in screen pixels, through
// the letterbox, for menus and games to point with. Input comes from a
// Source, so tests can play back a script.
package input

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Action is a logical input. Games add their own after the shared ones.
type Action int

const (
	Up Action = iota
	Down
	Left
	Right
	Confirm
	Back
	Pause
	Sound      // Ctrl+S
	Restart    // Ctrl+R
	Menu       // Ctrl+M, or the gamepad's Back/Select
	ScaleMode  // F9
	Quit       // F10
	Fullscreen // F11 or Alt+Enter
	NumActions
)

var actionNames = [NumActions]string{
	"Up", "Down", "Left", "Right", "Confirm", "Back", "Pause",
	"Sound", "Restart", "Menu", "ScaleMode", "Quit", "Fullscreen",
}

func (a Action) String() string {
	if a >= 0 && a < NumActions {
		return actionNames[a]
	}
	return "Action?"
}

// Button is a standard-layout gamepad button.
type Button = ebiten.StandardGamepadButton

// The standard-layout buttons the shared actions use.
const (
	PadA      = ebiten.StandardGamepadButtonRightBottom
	PadB      = ebiten.StandardGamepadButtonRightRight
	PadStart  = ebiten.StandardGamepadButtonCenterRight
	PadSelect = ebiten.StandardGamepadButtonCenterLeft
	PadUp     = ebiten.StandardGamepadButtonLeftTop
	PadDown   = ebiten.StandardGamepadButtonLeftBottom
	PadLeft   = ebiten.StandardGamepadButtonLeftLeft
	PadRight  = ebiten.StandardGamepadButtonLeftRight
)

// StickDeadZone is how far the left stick must be pushed to count as a
// direction.
const StickDeadZone = 0.5

// Frame is the raw input state for one tick: keys down, gamepad buttons
// down, and the left stick, with every connected gamepad combined; and the
// mouse: the pointer in window pixels, its buttons down, and the wheel.
type Frame struct {
	Keys           [ebiten.KeyMax + 1]bool
	Buttons        [ebiten.StandardGamepadButtonMax + 1]bool
	StickX, StickY float64 // -1–1; down and right are positive
	MouseX, MouseY float64
	Mouse          [ebiten.MouseButtonMax + 1]bool
	WheelY         float64 // positive away from the player
}

// Source fills in the raw input for the current tick.
type Source interface {
	Read(f *Frame)
}

// Ebiten reads the real keyboard and every standard-layout gamepad.
type Ebiten struct {
	keys []ebiten.Key
	pads []ebiten.GamepadID
}

// Read implements Source.
func (e *Ebiten) Read(f *Frame) {
	e.keys = inpututil.AppendPressedKeys(e.keys[:0])
	for _, k := range e.keys {
		if k >= 0 && k <= ebiten.KeyMax {
			f.Keys[k] = true
		}
	}
	e.pads = ebiten.AppendGamepadIDs(e.pads[:0])
	for _, id := range e.pads {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		for b := range f.Buttons {
			if ebiten.IsStandardGamepadButtonPressed(id, Button(b)) {
				f.Buttons[b] = true
			}
		}
		x := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		y := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
		if abs(x) > abs(f.StickX) {
			f.StickX = x
		}
		if abs(y) > abs(f.StickY) {
			f.StickY = y
		}
	}
	x, y := ebiten.CursorPosition()
	f.MouseX, f.MouseY = float64(x), float64(y)
	for b := range f.Mouse {
		f.Mouse[b] = ebiten.IsMouseButtonPressed(ebiten.MouseButton(b))
	}
	_, f.WheelY = ebiten.Wheel()
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// Menu repeat: a held direction repeats after RepeatDelay ticks, then every
// RepeatEvery ticks.
const (
	RepeatDelay = 20
	RepeatEvery = 5
)

// Input holds this tick's and last tick's state.
type Input struct {
	src        Source
	now, prev  Frame
	cur, last  []bool // per action
	held       []int  // ticks each action has been held
	extra      []Binding
	altEntered bool // Enter went down with Alt held; Confirm waits for its release
	view       View
}

// View is where the screen is shown in the window, for turning the pointer
// into screen pixels: the screen, Width×Height pixels, fills the rectangle
// at (X, Y) of size W×H in window pixels. The zero View is one window pixel
// per screen pixel, with no limit to the screen's size.
type View struct {
	X, Y, W, H    float64
	Width, Height int
}

// Binding maps a game-specific action to keys and buttons.
type Binding struct {
	Action  Action
	Keys    []ebiten.Key
	Buttons []Button
}

// New returns an Input reading from src. Bindings add game-specific actions
// numbered from NumActions up.
func New(src Source, bindings ...Binding) *Input {
	n := int(NumActions)
	for _, b := range bindings {
		n = max(n, int(b.Action)+1)
	}
	return &Input{src: src, cur: make([]bool, n), last: make([]bool, n), held: make([]int, n), extra: bindings}
}

// Update samples the input. Call once per tick.
func (in *Input) Update() {
	in.prev = in.now
	in.now = Frame{}
	in.src.Read(&in.now)
	copy(in.last, in.cur)

	k := &in.now.Keys
	b := &in.now.Buttons
	alt := k[ebiten.KeyAlt] || k[ebiten.KeyAltLeft] || k[ebiten.KeyAltRight]
	ctrl := k[ebiten.KeyControl] || k[ebiten.KeyControlLeft] || k[ebiten.KeyControlRight]
	enter := k[ebiten.KeyEnter] || k[ebiten.KeyNumpadEnter]
	if alt && enter && !in.wasDown(ebiten.KeyEnter) && !in.wasDown(ebiten.KeyNumpadEnter) {
		in.altEntered = true
	}
	if !enter {
		in.altEntered = false
	}
	// Letters held with Ctrl are commands, not movement.
	letter := func(key ebiten.Key) bool { return k[key] && !ctrl }

	set := func(a Action, v bool) { in.cur[a] = v }
	set(Up, k[ebiten.KeyArrowUp] || letter(ebiten.KeyW) || b[PadUp] || in.now.StickY < -StickDeadZone)
	set(Down, k[ebiten.KeyArrowDown] || letter(ebiten.KeyS) || b[PadDown] || in.now.StickY > StickDeadZone)
	set(Left, k[ebiten.KeyArrowLeft] || letter(ebiten.KeyA) || b[PadLeft] || in.now.StickX < -StickDeadZone)
	set(Right, k[ebiten.KeyArrowRight] || letter(ebiten.KeyD) || b[PadRight] || in.now.StickX > StickDeadZone)
	set(Confirm, ((enter && !alt && !in.altEntered) || k[ebiten.KeySpace]) || b[PadA])
	// The right mouse button is Esc on the mouse: Back and Pause.
	right := in.now.Mouse[ebiten.MouseButtonRight]
	set(Back, k[ebiten.KeyEscape] || k[ebiten.KeyBackspace] || b[PadB] || right)
	set(Pause, k[ebiten.KeyEscape] || letter(ebiten.KeyP) || b[PadStart] || right)
	set(Sound, ctrl && k[ebiten.KeyS])
	set(Restart, ctrl && k[ebiten.KeyR])
	set(Menu, (ctrl && k[ebiten.KeyM]) || b[PadSelect])
	set(ScaleMode, k[ebiten.KeyF9])
	set(Quit, k[ebiten.KeyF10])
	set(Fullscreen, k[ebiten.KeyF11] || (alt && enter))

	for _, bind := range in.extra {
		v := false
		for _, key := range bind.Keys {
			v = v || k[key]
		}
		for _, btn := range bind.Buttons {
			v = v || b[btn]
		}
		in.cur[bind.Action] = v
	}

	for a, v := range in.cur {
		if v {
			in.held[a]++
		} else {
			in.held[a] = 0
		}
	}
}

func (in *Input) wasDown(key ebiten.Key) bool { return in.prev.Keys[key] }

// Held reports whether the action is active this tick.
func (in *Input) Held(a Action) bool { return in.cur[a] }

// JustPressed reports whether the action became active this tick.
func (in *Input) JustPressed(a Action) bool { return in.cur[a] && !in.last[a] }

// JustReleased reports whether the action stopped this tick.
func (in *Input) JustReleased(a Action) bool { return !in.cur[a] && in.last[a] }

// Repeat reports a press, then repeats while the action is held, for moving
// through menus.
func (in *Input) Repeat(a Action) bool {
	h := in.held[a]
	return h == 1 || (h > RepeatDelay && (h-RepeatDelay)%RepeatEvery == 0)
}

// SetView sets where the screen is shown in the window, for Pointer. Call it
// before Update whenever the window or scale mode may have changed.
func (in *Input) SetView(v View) { in.view = v }

// Pointer is the mouse pointer in screen pixels, and whether it is over the
// screen (not the letterbox around it).
func (in *Input) Pointer() (p image.Point, ok bool) {
	return in.toScreen(in.now.MouseX, in.now.MouseY)
}

func (in *Input) toScreen(wx, wy float64) (image.Point, bool) {
	v := in.view
	x, y := wx, wy
	if v.W > 0 && v.H > 0 {
		x = (wx - v.X) * float64(v.Width) / v.W
		y = (wy - v.Y) * float64(v.Height) / v.H
	}
	p := image.Pt(int(math.Floor(x)), int(math.Floor(y)))
	ok := p.X >= 0 && p.Y >= 0 && (v.Width == 0 || p.X < v.Width) && (v.Height == 0 || p.Y < v.Height)
	return p, ok
}

// PointerMoved reports whether the pointer moved this tick.
func (in *Input) PointerMoved() bool {
	return in.now.MouseX != in.prev.MouseX || in.now.MouseY != in.prev.MouseY
}

// Clicked reports whether the left mouse button went down this tick.
func (in *Input) Clicked() bool {
	return in.now.Mouse[ebiten.MouseButtonLeft] && !in.prev.Mouse[ebiten.MouseButtonLeft]
}

// MouseHeld reports whether the left mouse button is down, for dragging.
func (in *Input) MouseHeld() bool { return in.now.Mouse[ebiten.MouseButtonLeft] }

// Wheel is how far the mouse wheel turned this tick: positive away from the
// player (up), negative towards.
func (in *Input) Wheel() float64 { return in.now.WheelY }

// AnyJustPressed reports whether any key, gamepad button or mouse button
// went down this tick, for "press any key". Modifiers, the window keys
// (F9–F11), Ctrl commands and Alt+Enter don't count.
func (in *Input) AnyJustPressed() bool {
	k := &in.now.Keys
	if k[ebiten.KeyControl] || k[ebiten.KeyControlLeft] || k[ebiten.KeyControlRight] {
		return false
	}
	alt := k[ebiten.KeyAlt] || k[ebiten.KeyAltLeft] || k[ebiten.KeyAltRight]
	for key := ebiten.Key(0); key <= ebiten.KeyMax; key++ {
		if !k[key] || in.prev.Keys[key] {
			continue
		}
		switch key {
		case ebiten.KeyAlt, ebiten.KeyAltLeft, ebiten.KeyAltRight,
			ebiten.KeyControl, ebiten.KeyControlLeft, ebiten.KeyControlRight,
			ebiten.KeyShift, ebiten.KeyShiftLeft, ebiten.KeyShiftRight,
			ebiten.KeyMeta, ebiten.KeyMetaLeft, ebiten.KeyMetaRight,
			ebiten.KeyF9, ebiten.KeyF10, ebiten.KeyF11:
			continue
		}
		if alt && (key == ebiten.KeyEnter || key == ebiten.KeyNumpadEnter) {
			continue
		}
		return true
	}
	for btn, down := range in.now.Buttons {
		if down && !in.prev.Buttons[btn] {
			return true
		}
	}
	for btn, down := range in.now.Mouse {
		if down && !in.prev.Mouse[btn] {
			return true
		}
	}
	return false
}

// Step is one entry of a Script: keys, buttons and mouse buttons held for
// Ticks ticks (at least one). Pointer, if set, moves the mouse pointer
// there, in window pixels, where it stays for the steps after.
type Step struct {
	Keys           []ebiten.Key
	Buttons        []Button
	StickX, StickY float64
	Pointer        *image.Point
	Mouse          []ebiten.MouseButton
	Wheel          float64
	Ticks          int
}

// Script plays back steps, one tick at a time, then nothing.
type Script struct {
	Steps []Step
	step  int
	tick  int
	ptr   image.Point
}

// Read implements Source.
func (s *Script) Read(f *Frame) {
	if s.step >= len(s.Steps) {
		return
	}
	st := s.Steps[s.step]
	for _, k := range st.Keys {
		f.Keys[k] = true
	}
	for _, b := range st.Buttons {
		f.Buttons[b] = true
	}
	f.StickX, f.StickY = st.StickX, st.StickY
	if st.Pointer != nil {
		s.ptr = *st.Pointer
	}
	f.MouseX, f.MouseY = float64(s.ptr.X), float64(s.ptr.Y)
	for _, b := range st.Mouse {
		f.Mouse[b] = true
	}
	f.WheelY = st.Wheel
	s.tick++
	if s.tick >= max(st.Ticks, 1) {
		s.step, s.tick = s.step+1, 0
	}
}

// Done reports whether every step has been played.
func (s *Script) Done() bool { return s.step >= len(s.Steps) }

// Press is a Step that holds keys for one tick.
func Press(keys ...ebiten.Key) Step { return Step{Keys: keys, Ticks: 1} }

// Idle is a Step with nothing held for n ticks.
func Idle(n int) Step { return Step{Ticks: n} }

// MoveTo is a Step that moves the pointer to (x, y), in window pixels, for
// one tick.
func MoveTo(x, y int) Step { return Step{Pointer: &image.Point{x, y}, Ticks: 1} }

// Click is a Step that moves the pointer to (x, y) and holds the left
// button for one tick.
func Click(x, y int) Step {
	return Step{Pointer: &image.Point{x, y}, Mouse: []ebiten.MouseButton{ebiten.MouseButtonLeft}, Ticks: 1}
}
