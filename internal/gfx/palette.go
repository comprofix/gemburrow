package gfx

import (
	"image/color"

	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/gfx/palettes"
)

// Palette is the game's palette, chosen by the project preset. Code names
// colours with the constants of that palette's package, e.g.
// resurrect64.Gold.
func Palette() palettes.Palette { return config.Project.Palette }

// RGBA returns colour i of the game's palette.
func RGBA(i int) color.RGBA { return config.Project.Palette.Colors[i] }
