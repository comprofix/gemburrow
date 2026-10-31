# Gem Burrow

Tunnel through the earth, collect jewels and treasure, and outwit the goblins.

You are a dwarf digging through a cavern. Clear every jewel to finish a level,
drop money bags on the goblins that chase you, and grab the gold before it
vanishes. Impatient goblins swell into cave trolls that dig straight for you.
Once all of a level's goblins are in, a glowing mushroom appears in the top-right
corner: take it, and for a short time they flee and you can eat them. There are Easy, Normal and Hard skills, and two looks to switch
between in Options: Smooth and Modern pixel art.

Created from [gameskeleton](https://github.com/comprofix/gameskeleton) v0.5.1.

## Download

Builds for Windows and Linux (64-bit) are on the
[Releases](https://github.com/comprofix/gemburrow/releases) page, with
`SHA256SUMS.txt` to check them against.

- **Windows:** unzip `gemburrow-vX.Y.Z-windows-amd64.zip` and run `gemburrow.exe`.
- **Linux:** extract `gemburrow-vX.Y.Z-linux-amd64.tar.gz` and run `./gemburrow`.
  It needs OpenGL, X11 and ALSA, which desktop systems already have.

## Controls

| Action | Keys |
|---|---|
| Move and dig | Arrow keys or WASD |
| Fire | Space |
| Pause, back | Esc |

The menus also work by mouse. Alt+Enter toggles fullscreen.

## Requirements

- Go 1.27 or later
- **Linux only:** cgo plus the X11/GL/ALSA development packages:
  `libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config`

## Build and run

```sh
go run ./cmd/gemburrow                 # run the game
go run -tags debug ./cmd/gemburrow     # with the debug overlay
go generate ./assets               # rebuild sprite sheets
go vet ./...
go test ./...
go build -o bin/ ./cmd/gemburrow
```

## Personal documents

`DESIGN.md`, `PLAN.md`, `AGENTS.md` and `CLAUDE.md` are private working documents
and agent instructions. `.gitignore` keeps them out of git.

## License

MIT. See [LICENSE](LICENSE).
