// Package ui has the shared menu widget and drawing helpers for panels and
// text.
package ui

import (
	"image"

	"github.com/comprofix/gameskeleton/internal/gfx"
	"github.com/comprofix/gameskeleton/internal/input"
)

// Item is one line of a Menu.
type Item struct {
	Label    string
	Value    func() string // optional: shown after the label, e.g. "On"
	Disabled bool
	// DisabledIf, if set, disables the item while it returns true, e.g. a
	// volume while its sound is switched off.
	DisabledIf func() bool
	// Slider, if set, makes the item a slider: a bar filled to its 0–1 value
	// and the value as a percentage, in place of Value.
	Slider func() float64
	// SetSlider, if set, sets a slider's value (0–1) when the bar is
	// clicked or dragged with the mouse.
	SetSlider func(v float64)
	Confirm   func()        // Confirm on this item
	Adjust    func(dir int) // Left (-1) or Right (+1) on this item
}

// IsDisabled reports whether the item can't be chosen right now.
func (it Item) IsDisabled() bool {
	return it.Disabled || (it.DisabledIf != nil && it.DisabledIf())
}

// Menu is a vertical list of items with a cursor. Up and Down move (with
// repeat, wrapping and skipping disabled items), Confirm activates, and
// Left/Right adjust values. Once placed, it works by mouse too: the pointer
// moves the cursor to the item under it, a click activates that item as
// Confirm does (a click on a slider's bar sets it, and dragging moves it),
// and the wheel moves the cursor.
type Menu struct {
	Items  []Item
	Cursor int
	// OnMove and OnConfirm are called after the cursor moves or an item is
	// activated, for sounds.
	OnMove    func()
	OnConfirm func()
	// Rows, if set, is how many items show at once: a longer menu scrolls
	// to keep the cursor in view.
	Rows int

	top      int      // the first item shown, while it scrolls
	font     gfx.Font // where it is drawn, for the mouse; nil until placed
	at       image.Point
	dragging bool // a slider's bar is held with the mouse
}

// Place records where the menu is drawn, with its top-left corner at (x, y)
// in font, so the mouse can point at its items. Call it before Update, with
// what Draw is given. Without it, or with a nil font, the menu ignores the
// mouse.
func (m *Menu) Place(font gfx.Font, x, y int) {
	m.font, m.at = font, image.Pt(x, y)
}

// PlaceCentred is Place for a menu drawn by DrawCentred on a screen
// screenW wide.
func (m *Menu) PlaceCentred(font gfx.Font, screenW, y int) {
	if font == nil {
		m.font = nil
		return
	}
	w, _ := m.Size(font)
	m.Place(font, centredX(screenW, w), y)
}

// Update handles one tick of input.
func (m *Menu) Update(in *input.Input) {
	if len(m.Items) == 0 {
		return
	}
	defer func() { m.top = m.First() }()
	m.skipDisabled(1)
	if m.mouse(in) {
		return
	}
	switch {
	case in.Repeat(input.Up):
		m.move(-1)
	case in.Repeat(input.Down):
		m.move(1)
	case in.JustPressed(input.Confirm):
		m.confirm()
	case in.Repeat(input.Left):
		m.adjust(-1)
	case in.Repeat(input.Right):
		m.adjust(1)
	}
}

// confirm activates the item under the cursor.
func (m *Menu) confirm() {
	it := m.Items[m.Cursor]
	if it.IsDisabled() {
		return
	}
	if it.Confirm != nil {
		it.Confirm()
	} else if it.Adjust != nil {
		it.Adjust(1)
	}
	if m.OnConfirm != nil {
		m.OnConfirm()
	}
}

// mouse handles the mouse for one tick and reports whether it did
// anything, so the keys wait for the next tick.
func (m *Menu) mouse(in *input.Input) bool {
	if m.font == nil {
		m.dragging = false
		return false
	}
	p, over := in.Pointer()
	if m.dragging {
		if in.MouseHeld() {
			if in.PointerMoved() {
				m.slide(p.X)
			}
			return true
		}
		m.dragging = false
	}
	if w := in.Wheel(); w != 0 {
		if w > 0 {
			m.move(-1)
		} else {
			m.move(1)
		}
		return true
	}
	if !over {
		return false
	}
	i := m.itemAt(p)
	if i < 0 || m.Items[i].IsDisabled() {
		return false
	}
	moved := i != m.Cursor && (in.PointerMoved() || in.Clicked())
	if moved {
		m.Cursor = i
		if m.OnMove != nil {
			m.OnMove()
		}
	}
	if !in.Clicked() {
		return moved
	}
	if it := m.Items[i]; it.Slider != nil {
		// A click on the bar sets it; elsewhere on the line only moves
		// the cursor.
		if it.SetSlider != nil && m.onBar(p) {
			m.dragging = true
			m.slide(p.X)
		}
		return true
	}
	m.confirm()
	return true
}

// itemAt is the item whose line is under p, in screen pixels, or -1.
func (m *Menu) itemAt(p image.Point) int {
	w, h := m.Size(m.font)
	if p.X < m.at.X || p.X >= m.at.X+w || p.Y < m.at.Y || p.Y >= m.at.Y+h+LineGap {
		return -1
	}
	r := (p.Y - m.at.Y) / (m.font.LineHeight() + LineGap)
	return m.First() + min(r, m.Shown()-1)
}

// bar is the x of the cursor item's slider bar, and its width.
func (m *Menu) bar() (x, w int) {
	cell := m.font.CellWidth()
	return m.at.X + 2*cell + m.sliderColumn(m.font), SliderCells * cell
}

// onBar reports whether x is on the cursor item's slider bar, give or take
// half a cell, so a click at either end reaches 0 or 1.
func (m *Menu) onBar(p image.Point) bool {
	x, w := m.bar()
	half := m.font.CellWidth() / 2
	return p.X >= x-half && p.X < x+w+half
}

// slide sets the cursor item's slider to where x lies along its bar, as
// the bar is filled: from two pixels in at each end.
func (m *Menu) slide(x int) {
	it := m.Items[m.Cursor]
	if it.SetSlider == nil || it.IsDisabled() {
		return
	}
	bx, w := m.bar()
	v := min(max(float64(x-bx-2)/float64(w-4), 0), 1)
	before := it.Slider()
	it.SetSlider(v)
	if it.Slider() != before && m.OnMove != nil {
		m.OnMove()
	}
}

func (m *Menu) move(dir int) {
	if m.step(dir) && m.OnMove != nil {
		m.OnMove()
	}
}

// step moves the cursor to the next enabled item in direction dir and
// reports whether it moved.
func (m *Menu) step(dir int) bool {
	start := m.Cursor
	for range m.Items {
		m.Cursor = (m.Cursor + dir + len(m.Items)) % len(m.Items)
		if !m.Items[m.Cursor].IsDisabled() {
			break
		}
	}
	return m.Cursor != start
}

func (m *Menu) adjust(dir int) {
	it := m.Items[m.Cursor]
	if it.IsDisabled() || it.Adjust == nil {
		return
	}
	it.Adjust(dir)
	if m.OnMove != nil {
		m.OnMove()
	}
}

// skipDisabled moves the cursor off a disabled item, if it is on one,
// silently.
func (m *Menu) skipDisabled(dir int) {
	m.Cursor = min(max(m.Cursor, 0), len(m.Items)-1)
	if m.Items[m.Cursor].IsDisabled() {
		m.step(dir)
	}
}

// Scrolls reports whether the menu has more items than it shows at once.
func (m *Menu) Scrolls() bool { return m.Rows > 0 && len(m.Items) > m.Rows }

// Shown is how many items show at once.
func (m *Menu) Shown() int {
	if !m.Scrolls() {
		return len(m.Items)
	}
	return m.Rows
}

// First is the first item shown: the top one, unless the menu scrolls;
// then the one that keeps the cursor in view, moving as little as it can
// from where it was.
func (m *Menu) First() int {
	if !m.Scrolls() {
		return 0
	}
	top := min(max(m.top, m.Cursor-m.Rows+1), m.Cursor)
	return min(max(top, 0), len(m.Items)-m.Rows)
}

// Text returns item i's text: the label, then its value if any. A slider's
// value is its percentage.
func (m *Menu) Text(i int) string {
	it := m.Items[i]
	if it.Slider != nil {
		return it.Label + ": " + percent(min(max(it.Slider(), 0), 1))
	}
	if it.Value == nil {
		return it.Label
	}
	return it.Label + ": " + it.Value()
}

// OnOff is a Value for a boolean setting.
func OnOff(v *bool) func() string {
	return func() string {
		if *v {
			return "On"
		}
		return "Off"
	}
}
