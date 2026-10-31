package config

import "fmt"

// Looks are a game's looks: the first, which players start on, and the
// second, if the game has one. Two looks are a pixel look and a smooth one
// drawing the same game, the smooth screen a whole multiple of the pixel
// one, so the game's coordinates are the same in both (see Check).
type Looks struct {
	First  Preset
	Second *Preset // nil for a game with one look
}

// ProjectLooks are the looks chosen in project.go.
func ProjectLooks() Looks { return Looks{First: Project, Second: Second} }

// Two reports whether there is a second look.
func (l Looks) Two() bool { return l.Second != nil }

// Named is the look called name, or the first for any other name.
func (l Looks) Named(name string) Preset {
	if l.Second != nil && name == l.Second.Name {
		return *l.Second
	}
	return l.First
}

// Pixel is the pixel look of two, or nil in a game with one look.
func (l Looks) Pixel() *Preset {
	switch {
	case l.Second == nil:
		return nil
	case l.Second.Art == ArtPixel:
		return l.Second
	}
	return &l.First
}

// scaled is the look whose scale modes the saved scale mode is one of: the
// pixel look of two, or the only one.
func (l Looks) scaled() Preset {
	if p := l.Pixel(); p != nil {
		return *p
	}
	return l.First
}

// Check reports whether the looks can be shown in one game. A second look
// must be in the other art style, with another name (the settings save it),
// the same palette and audio style, and a screen a whole multiple of the
// pixel look's in both directions.
func (l Looks) Check() error {
	if l.Second == nil {
		return nil
	}
	a, b := l.First, *l.Second
	switch {
	case a.Name == b.Name:
		return fmt.Errorf("both looks are named %q", a.Name)
	case a.Art == b.Art:
		return fmt.Errorf("looks %s and %s are both %s art: one must be pixel and one smooth", a.Name, b.Name, a.Art)
	case a.Palette.Name != b.Palette.Name:
		return fmt.Errorf("looks %s and %s have different palettes (%s, %s)", a.Name, b.Name, a.Palette.Name, b.Palette.Name)
	case a.Audio != b.Audio:
		return fmt.Errorf("looks %s and %s have different audio styles (%s, %s)", a.Name, b.Name, a.Audio, b.Audio)
	}
	px, sm := a, b
	if px.Art != ArtPixel {
		px, sm = b, a
	}
	k := sm.ScreenWidth / px.ScreenWidth
	if k < 1 || sm.ScreenWidth != k*px.ScreenWidth || sm.ScreenHeight != k*px.ScreenHeight {
		return fmt.Errorf("look %s (%d×%d) is not a whole multiple of %s (%d×%d)",
			sm.Name, sm.ScreenWidth, sm.ScreenHeight, px.Name, px.ScreenWidth, px.ScreenHeight)
	}
	return nil
}
