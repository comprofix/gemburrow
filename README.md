# Gem Burrow

Tunnel through the earth, collect jewels and treasure, and outwit the goblins.

Created from [gameskeleton](https://github.com/comprofix/gameskeleton) v0.5.1.

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
