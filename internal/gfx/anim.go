package gfx

// Animator steps through a sprite's frames by ticks.
type Animator struct {
	TicksPerFrame int // ticks each frame is shown
	Frames        int // number of frames to cycle
	frame         int
	elapsed       int
}

// Advance moves the animation on by the given number of ticks.
func (a *Animator) Advance(ticks int) {
	if a.TicksPerFrame <= 0 || a.Frames <= 0 {
		return
	}
	a.elapsed += ticks
	for a.elapsed >= a.TicksPerFrame {
		a.elapsed -= a.TicksPerFrame
		a.frame = (a.frame + 1) % a.Frames
	}
}

// Frame returns the current frame index.
func (a *Animator) Frame() int { return a.frame }

// Reset returns to the first frame.
func (a *Animator) Reset() { a.frame, a.elapsed = 0, 0 }
