// Package assets embeds the game's generated sprite sheets and fonts.
//
// Sprite sources live in sprites/*.txt; regenerate the sheets with:
//
//	go generate ./assets
package assets

import "embed"

//go:generate go run ../tools/spritegen -in sprites -out gen

// FS holds the generated sheets (gen/*.png, gen/*.json) and the fonts.
//
//go:embed gen font
var FS embed.FS
