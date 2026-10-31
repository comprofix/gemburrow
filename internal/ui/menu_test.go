package ui

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/gfx"
	"github.com/comprofix/gameskeleton/internal/input"
)

func feed(steps ...input.Step) *input.Input {
	return input.New(&input.Script{Steps: steps})
}

func TestMenuMovesAndSkipsDisabled(t *testing.T) {
	m := &Menu{Items: []Item{{Label: "A"}, {Label: "B", Disabled: true}, {Label: "C"}}}
	moves := 0
	m.OnMove = func() { moves++ }
	in := feed(input.Press(ebiten.KeyArrowDown), input.Idle(1), input.Press(ebiten.KeyArrowDown), input.Idle(1), input.Press(ebiten.KeyArrowUp))
	want := []int{2, 2, 0, 0, 2} // B is skipped; the menu wraps both ways
	for i, w := range want {
		in.Update()
		m.Update(in)
		if m.Cursor != w {
			t.Fatalf("tick %d: cursor %d, want %d", i, m.Cursor, w)
		}
	}
	if moves != 3 {
		t.Errorf("OnMove called %d times, want 3", moves)
	}
}

func TestMenuStartsOffADisabledItem(t *testing.T) {
	m := &Menu{Items: []Item{{Label: "A", Disabled: true}, {Label: "B"}}}
	in := feed(input.Idle(1))
	in.Update()
	m.Update(in)
	if m.Cursor != 1 {
		t.Errorf("cursor %d, want 1", m.Cursor)
	}
}

func TestMenuConfirmAndAdjust(t *testing.T) {
	confirmed, level := 0, 5
	on := false
	m := &Menu{Items: []Item{
		{Label: "Go", Confirm: func() { confirmed++ }},
		{Label: "Level", Value: func() string { return string(rune('0' + level)) }, Adjust: func(d int) { level += d }},
		{Label: "Flag", Value: OnOff(&on), Adjust: func(int) { on = !on }},
	}}
	in := feed(
		input.Press(ebiten.KeyEnter), input.Idle(1),
		input.Press(ebiten.KeyArrowDown), input.Idle(1),
		input.Press(ebiten.KeyArrowRight), input.Idle(1),
		input.Press(ebiten.KeyArrowRight), input.Idle(1),
		input.Press(ebiten.KeyArrowLeft), input.Idle(1),
		input.Press(ebiten.KeyArrowDown), input.Idle(1),
		input.Press(ebiten.KeyEnter), // Confirm on an Adjust-only item adjusts it
	)
	for range 13 {
		in.Update()
		m.Update(in)
	}
	if confirmed != 1 {
		t.Errorf("confirmed %d times, want 1", confirmed)
	}
	if level != 6 {
		t.Errorf("level %d, want 6", level)
	}
	if !on || m.Text(2) != "Flag: On" {
		t.Errorf("flag %v, text %q", on, m.Text(2))
	}
	if m.Text(0) != "Go" {
		t.Errorf("text %q", m.Text(0))
	}
}

func TestMenuIgnoresDisabledConfirm(t *testing.T) {
	called := false
	m := &Menu{Items: []Item{{Label: "X", Disabled: true, Confirm: func() { called = true }}}}
	in := feed(input.Press(ebiten.KeyEnter))
	in.Update()
	m.Update(in)
	if called {
		t.Error("a disabled item was activated")
	}
}

func TestBlink(t *testing.T) {
	if !Blink(0, 40) || !Blink(19, 40) || Blink(20, 40) || Blink(39, 40) || !Blink(40, 40) {
		t.Error("Blink should be on for the first half of each period")
	}
}

func TestSkippingADisabledItemIsSilent(t *testing.T) {
	m := &Menu{Items: []Item{{Label: "A", Disabled: true}, {Label: "B"}}}
	m.OnMove = func() { t.Error("OnMove called when skipping a disabled first item") }
	in := feed(input.Idle(1))
	in.Update()
	m.Update(in)
}

func TestDisabledIfIsSkippedAndSliderText(t *testing.T) {
	on, v := false, 0.6
	m := &Menu{Items: []Item{
		{Label: "Music", Adjust: func(int) { on = !on }},
		{Label: "Music volume", Slider: func() float64 { return v }, DisabledIf: func() bool { return !on },
			Adjust: func(d int) { v += 0.1 * float64(d) }},
		{Label: "Back"},
	}}
	in := feed(input.Press(ebiten.KeyArrowDown), input.Idle(1), input.Press(ebiten.KeyArrowUp), input.Idle(1),
		input.Press(ebiten.KeyEnter), input.Idle(1), input.Press(ebiten.KeyArrowDown))
	want := []int{2, 2, 0, 0, 0, 0, 1} // off: the slider is skipped; on: it is not
	for i, w := range want {
		in.Update()
		m.Update(in)
		if m.Cursor != w {
			t.Fatalf("tick %d: cursor %d, want %d", i, m.Cursor, w)
		}
	}
	if got := m.Text(1); got != "Music volume: 60%" {
		t.Errorf("slider text %q", got)
	}
}

// TestMenuScrolls: a menu with more items than Rows shows Rows at a time,
// moving the window only as far as it must to keep the cursor in view,
// both ways and when the cursor wraps.
func TestMenuScrolls(t *testing.T) {
	m := &Menu{Rows: 3}
	for range 6 {
		m.Items = append(m.Items, Item{Label: "X"})
	}
	var steps []input.Step
	for _, k := range []ebiten.Key{
		ebiten.KeyArrowDown, ebiten.KeyArrowDown, ebiten.KeyArrowDown, ebiten.KeyArrowDown, // to 4
		ebiten.KeyArrowUp, ebiten.KeyArrowUp, ebiten.KeyArrowUp, // back to 1
		ebiten.KeyArrowUp, ebiten.KeyArrowUp, // to 0, then round to 5
	} {
		steps = append(steps, input.Press(k), input.Idle(1))
	}
	in := feed(steps...)
	want := []struct{ cursor, first int }{
		{1, 0}, {2, 0}, {3, 1}, {4, 2},
		{3, 2}, {2, 2}, {1, 1},
		{0, 0}, {5, 3},
	}
	for i, w := range want {
		for range 2 {
			in.Update()
			m.Update(in)
		}
		if m.Cursor != w.cursor || m.First() != w.first {
			t.Fatalf("move %d: cursor %d, first %d; want %d, %d", i, m.Cursor, m.First(), w.cursor, w.first)
		}
	}
	if !m.Scrolls() || m.Shown() != 3 {
		t.Errorf("scrolls %v, shows %d: want 3 of 6", m.Scrolls(), m.Shown())
	}
	short := &Menu{Rows: 8, Items: m.Items}
	if short.Scrolls() || short.Shown() != 6 || short.First() != 0 {
		t.Errorf("a menu shorter than Rows: scrolls %v, shows %d", short.Scrolls(), short.Shown())
	}
}

// mouseFont is the font the mouse tests place menus in: cells 6 wide, lines
// 12 tall, so rows are 16 apart.
var mouseFont = gfx.NewVectorFont(10)

// rowY is the middle of row r of a menu placed at y in mouseFont.
func rowY(y, r int) int { return y + r*(mouseFont.LineHeight()+LineGap) + mouseFont.LineHeight()/2 }

func TestMenuMouse(t *testing.T) {
	chosen := ""
	m := &Menu{Items: []Item{
		{Label: "Apple", Confirm: func() { chosen = "Apple" }},
		{Label: "Banana", Disabled: true},
		{Label: "Cherry", Confirm: func() { chosen = "Cherry" }},
	}}
	moves, confirms := 0, 0
	m.OnMove = func() { moves++ }
	m.OnConfirm = func() { confirms++ }
	const x, y = 100, 50
	in := feed(
		input.MoveTo(x+10, rowY(y, 2)), // hover moves the cursor
		input.MoveTo(x+20, rowY(y, 1)), // not onto a disabled item
		input.Click(x+10, rowY(y, 0)),  // a click moves and chooses
		input.Idle(1),
		input.Click(x-5, rowY(y, 2)), // beside the menu: nothing
		input.Idle(1),
		input.Step{Wheel: -1, Ticks: 1}, // the wheel moves the cursor, skipping
		input.Press(ebiten.KeyArrowUp),  // keys still work with the pointer at rest
	)
	type state struct {
		cursor int
		chosen string
	}
	want := []state{{2, ""}, {2, ""}, {0, "Apple"}, {0, "Apple"}, {0, "Apple"}, {0, "Apple"}, {2, "Apple"}, {0, "Apple"}}
	for i, w := range want {
		in.Update()
		m.Place(mouseFont, x, y)
		m.Update(in)
		if got := (state{m.Cursor, chosen}); got != w {
			t.Fatalf("tick %d: %+v, want %+v", i, got, w)
		}
	}
	if moves != 4 || confirms != 1 {
		t.Errorf("OnMove %d times and OnConfirm %d, want 4 and 1", moves, confirms)
	}
}

func TestMenuIgnoresTheMouseUntilPlaced(t *testing.T) {
	chosen := false
	m := &Menu{Items: []Item{{Label: "Go", Confirm: func() { chosen = true }}}}
	in := feed(input.Click(5, 5))
	in.Update()
	m.Update(in)
	if chosen {
		t.Error("an unplaced menu took a click")
	}
}

// TestMenuSliderByMouse: a click on a slider's bar sets it, dragging moves
// it, and a click on its label only moves the cursor.
func TestMenuSliderByMouse(t *testing.T) {
	v := 0.2
	m := &Menu{Items: []Item{
		{Label: "Back"},
		{Label: "Vol", Slider: func() float64 { return v }, SetSlider: func(x float64) { v = x }},
	}}
	const x, y = 100, 50
	cell := mouseFont.CellWidth()
	bx := x + 2*cell + mouseFont.Width("Vol") + cell // the bar's left edge
	bw := SliderCells * cell
	held := func(px int) input.Step {
		return input.Step{Pointer: &image.Point{px, rowY(y, 1)}, Mouse: []ebiten.MouseButton{ebiten.MouseButtonLeft}, Ticks: 1}
	}
	in := feed(
		input.Click(x+cell, rowY(y, 1)), // the label: v unchanged
		input.Idle(1),
		input.Click(bx+2+(bw-4)/2, rowY(y, 1)), // the middle of the bar
		held(bx+bw+cell),                       // dragged past the end
		held(bx-cell),                          // and back past the start
		input.MoveTo(bx+bw/2, rowY(y, 1)),      // let go: moving no longer sets it
	)
	want := []float64{0.2, 0.2, 0.5, 1, 0, 0}
	for i, w := range want {
		in.Update()
		m.Place(mouseFont, x, y)
		m.Update(in)
		if v != w || m.Cursor != 1 {
			t.Fatalf("tick %d: value %v, cursor %d; want %v, 1", i, v, m.Cursor, w)
		}
	}
}
