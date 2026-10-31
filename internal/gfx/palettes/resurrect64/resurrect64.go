// Package resurrect64 is the Resurrect 64 palette by Kerrie Lake
// (lospec.com/palette-list/resurrect-64), used by the Modern preset.
// The palette has no official colour names; these are our
// own, in the palette's published order.
package resurrect64

import "github.com/comprofix/gameskeleton/internal/gfx/palettes"

// Colour indices.
const (
	Ink = iota
	Shadow
	Dusk
	Clay
	Sand
	Mauve
	Lilac
	Fog
	Mist
	White
	Maroon
	Brick
	Vermilion
	Coral
	Crimson
	Red
	Orange
	Amber
	Gold
	Wine
	Rust
	Copper
	Caramel
	Honey
	Bark
	Olive
	Moss
	Chartreuse
	Lemon
	Pine
	Forest
	Emerald
	Leaf
	Lime
	Charcoal
	Slate
	Sage
	Fern
	Khaki
	DeepTeal
	Teal
	Jade
	Aqua
	Seafoam
	Navy
	Indigo
	Cobalt
	Blue
	Sky
	Plum
	Grape
	Violet
	Lavender
	Orchid
	Berry
	Raspberry
	Rose
	Pink
	Mulberry
	Cerise
	Flamingo
	Salmon
	Apricot
	Blush
)

// Palette is Resurrect 64.
var Palette = palettes.New("Resurrect 64", []palettes.Entry{
	Ink:        {Name: "Ink", Hex: 0x2e222f},
	Shadow:     {Name: "Shadow", Hex: 0x3e3546},
	Dusk:       {Name: "Dusk", Hex: 0x625565},
	Clay:       {Name: "Clay", Hex: 0x966c6c},
	Sand:       {Name: "Sand", Hex: 0xab947a},
	Mauve:      {Name: "Mauve", Hex: 0x694f62},
	Lilac:      {Name: "Lilac", Hex: 0x7f708a},
	Fog:        {Name: "Fog", Hex: 0x9babb2},
	Mist:       {Name: "Mist", Hex: 0xc7dcd0},
	White:      {Name: "White", Hex: 0xffffff},
	Maroon:     {Name: "Maroon", Hex: 0x6e2727},
	Brick:      {Name: "Brick", Hex: 0xb33831},
	Vermilion:  {Name: "Vermilion", Hex: 0xea4f36},
	Coral:      {Name: "Coral", Hex: 0xf57d4a},
	Crimson:    {Name: "Crimson", Hex: 0xae2334},
	Red:        {Name: "Red", Hex: 0xe83b3b},
	Orange:     {Name: "Orange", Hex: 0xfb6b1d},
	Amber:      {Name: "Amber", Hex: 0xf79617},
	Gold:       {Name: "Gold", Hex: 0xf9c22b},
	Wine:       {Name: "Wine", Hex: 0x7a3045},
	Rust:       {Name: "Rust", Hex: 0x9e4539},
	Copper:     {Name: "Copper", Hex: 0xcd683d},
	Caramel:    {Name: "Caramel", Hex: 0xe6904e},
	Honey:      {Name: "Honey", Hex: 0xfbb954},
	Bark:       {Name: "Bark", Hex: 0x4c3e24},
	Olive:      {Name: "Olive", Hex: 0x676633},
	Moss:       {Name: "Moss", Hex: 0xa2a947},
	Chartreuse: {Name: "Chartreuse", Hex: 0xd5e04b},
	Lemon:      {Name: "Lemon", Hex: 0xfbff86},
	Pine:       {Name: "Pine", Hex: 0x165a4c},
	Forest:     {Name: "Forest", Hex: 0x239063},
	Emerald:    {Name: "Emerald", Hex: 0x1ebc73},
	Leaf:       {Name: "Leaf", Hex: 0x91db69},
	Lime:       {Name: "Lime", Hex: 0xcddf6c},
	Charcoal:   {Name: "Charcoal", Hex: 0x313638},
	Slate:      {Name: "Slate", Hex: 0x374e4a},
	Sage:       {Name: "Sage", Hex: 0x547e64},
	Fern:       {Name: "Fern", Hex: 0x92a984},
	Khaki:      {Name: "Khaki", Hex: 0xb2ba90},
	DeepTeal:   {Name: "DeepTeal", Hex: 0x0b5e65},
	Teal:       {Name: "Teal", Hex: 0x0b8a8f},
	Jade:       {Name: "Jade", Hex: 0x0eaf9b},
	Aqua:       {Name: "Aqua", Hex: 0x30e1b9},
	Seafoam:    {Name: "Seafoam", Hex: 0x8ff8e2},
	Navy:       {Name: "Navy", Hex: 0x323353},
	Indigo:     {Name: "Indigo", Hex: 0x484a77},
	Cobalt:     {Name: "Cobalt", Hex: 0x4d65b4},
	Blue:       {Name: "Blue", Hex: 0x4d9be6},
	Sky:        {Name: "Sky", Hex: 0x8fd3ff},
	Plum:       {Name: "Plum", Hex: 0x45293f},
	Grape:      {Name: "Grape", Hex: 0x6b3e75},
	Violet:     {Name: "Violet", Hex: 0x905ea9},
	Lavender:   {Name: "Lavender", Hex: 0xa884f3},
	Orchid:     {Name: "Orchid", Hex: 0xeaaded},
	Berry:      {Name: "Berry", Hex: 0x753c54},
	Raspberry:  {Name: "Raspberry", Hex: 0xa24b6f},
	Rose:       {Name: "Rose", Hex: 0xcf657f},
	Pink:       {Name: "Pink", Hex: 0xed8099},
	Mulberry:   {Name: "Mulberry", Hex: 0x831c5d},
	Cerise:     {Name: "Cerise", Hex: 0xc32454},
	Flamingo:   {Name: "Flamingo", Hex: 0xf04f78},
	Salmon:     {Name: "Salmon", Hex: 0xf68181},
	Apricot:    {Name: "Apricot", Hex: 0xfca790},
	Blush:      {Name: "Blush", Hex: 0xfdcbb0},
})
