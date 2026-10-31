// Package config holds the project settings, the player's saved settings
// and the game's tuning values.
package config

// Title is the game's name, shown in the window title and on the title
// screen.
const Title = "Game Skeleton"

// TPS is the game loop rate. Game logic counts in ticks of 1/TPS seconds.
const TPS = 60

// Project is the preset this game uses: its resolution, scale modes and
// palette. This is the only place they are chosen. In a game with two
// looks it is the first, which players start on.
var Project = Modern

// Second is the game's other look, or nil for a game with one. The pair
// must pass Looks.Check: a pixel look and a smooth one, with the same
// palette and audio style, the smooth screen a whole multiple of the pixel
// one.
var Second *Preset
