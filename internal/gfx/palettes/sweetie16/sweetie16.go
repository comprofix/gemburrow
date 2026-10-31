// Package sweetie16 is the Sweetie 16 palette by GrafxKid
// (lospec.com/palette-list/sweetie-16), used by the Classic PC preset.
package sweetie16

import "github.com/comprofix/gameskeleton/internal/gfx/palettes"

// Colour indices.
const (
	Night = iota
	Plum
	Red
	Orange
	Yellow
	Lime
	Green
	Teal
	Navy
	Blue
	Sky
	Cyan
	White
	Silver
	Slate
	Charcoal
)

// Palette is Sweetie 16.
var Palette = palettes.New("Sweetie 16", []palettes.Entry{
	Night:    {Name: "Night", Hex: 0x1a1c2c},
	Plum:     {Name: "Plum", Hex: 0x5d275d},
	Red:      {Name: "Red", Hex: 0xb13e53},
	Orange:   {Name: "Orange", Hex: 0xef7d57},
	Yellow:   {Name: "Yellow", Hex: 0xffcd75},
	Lime:     {Name: "Lime", Hex: 0xa7f070},
	Green:    {Name: "Green", Hex: 0x38b764},
	Teal:     {Name: "Teal", Hex: 0x257179},
	Navy:     {Name: "Navy", Hex: 0x29366f},
	Blue:     {Name: "Blue", Hex: 0x3b5dc9},
	Sky:      {Name: "Sky", Hex: 0x41a6f6},
	Cyan:     {Name: "Cyan", Hex: 0x73eff7},
	White:    {Name: "White", Hex: 0xf4f4f4},
	Silver:   {Name: "Silver", Hex: 0x94b0c2},
	Slate:    {Name: "Slate", Hex: 0x566c86},
	Charcoal: {Name: "Charcoal", Hex: 0x333c57},
})
