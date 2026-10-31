package gfx

import "github.com/comprofix/gameskeleton/internal/gfx/palettes"

// Scheme gives the shared scenes and menus their colours by role, so they
// look right with any palette. Each role is a palette index.
type Scheme struct {
	Backdrop  int // screen background
	Panel     int // menu and dialog panels
	Border    int // panel borders
	Text      int // normal text
	Dim       int // disabled or secondary text
	Highlight int // the selected menu item
	Title     int // titles
	Shadow    int // text shadows and fades
}

// SchemeFor picks each role's colour by name, taking the first name the
// palette has, and falls back to the darkest or brightest colour. The names
// cover the ready-made palettes; a game can set its own Scheme instead.
func SchemeFor(p palettes.Palette) Scheme {
	dark, bright := p.Darkest(), p.Brightest()
	pick := func(fallback int, names ...string) int {
		for _, n := range names {
			if i, ok := p.Lookup(n); ok {
				return i
			}
		}
		return fallback
	}
	return Scheme{
		Backdrop:  pick(dark, "Navy"),
		Panel:     pick(dark, "Shadow", "Charcoal"),
		Border:    pick(bright, "Fog", "Silver"),
		Text:      bright,
		Dim:       pick(bright, "Lilac", "Slate"),
		Highlight: pick(bright, "Gold", "Yellow"),
		Title:     pick(bright, "Gold", "Yellow"),
		Shadow:    dark,
	}
}
