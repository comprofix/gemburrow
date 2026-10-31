# Game Skeleton

A starter template for 2D games written in Go with
[Ebitengine](https://ebitengine.org/), in one of two art styles: **pixel
art**, for retro and arcade games, or **smooth**, art drawn in code at
1280×720, for card, board and table games; or both, as two looks the player
switches between in Options. New games are made from it with
GitHub's **Use this template** button. It comes with a tiny pixel-art demo
game, **Dot Dash**, which shows every part working and is removed when a
real game begins.

## What you get

- **Display:** a fixed internal resolution, letterboxed, with fullscreen.
  Pixel art is scaled up by whole numbers, with an optional CRT 4:3 mode;
  smooth art is scaled smoothly to any size. One palette per game, with
  colours named in code (`resurrect64.Gold`).
- **Pixel art:** text-grid sprites in `assets/sprites/*.txt`, packed into
  sprite sheets by `go generate`, with a preview sheet for checking them by
  eye. An 8×8 font and a 16×16 font made from it.
- **Smooth art:** shapes drawn in code in palette colours (rounded
  rectangles, circles, lines, polygons, panels and buttons with hover and
  pressed looks), and Go Mono Bold for text. Sprites work here too.
- **Two looks:** a game can offer a pixel look and a smooth one, switched
  in Options. The menus are drawn for both; the game's own scenes start as
  pixel art, enlarged in the smooth look, and can be redrawn one at a time.
- **Program icon:** drawn as a sprite like the rest, shown on the window and
  linked into the Windows exe. The skeleton's is a placeholder for each game
  to replace with its own.
- **Scenes:** title, main menu, skill select, options, pause, tally, high
  score entry and table, with fades and screen shake.
- **Input:** keyboard, gamepad and mouse mapped to actions, with key repeat
  for menus. Menus and the shared scenes work by mouse too: pointing moves
  the bar, a click chooses, the wheel scrolls.
- **Sound made in code:** a chip synth with pulse, triangle and noise waves and
  volume envelopes, tunes and effects as note tables, timed to the game tick.
- **Saves:** settings and a top-10 high score table for each skill.
- **Testing:** game logic runs headless and deterministic from a seed;
  scripted input, a scene harness and golden replay tests. A debug build shows
  a timing overlay and collision boxes.
- **Agent-neutral setup:** the skeleton's `AGENTS.md` contains shared project
  rules and the new-game setup workflow. Generated games inherit the shared
  rules in their own tracked `AGENTS.md`; `CLAUDE.md` is a compatibility
  hand-off for Claude Code.

## Requirements

- Go 1.27 or later
- **Linux only:** cgo plus the X11/GL/ALSA development packages:
  `libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config`

## Presets

A game picks one preset in `internal/config/project.go`, along with its title
and tick rate. Nothing else in the code hard-codes a screen size or palette.

| | Modern (pixel, default) | Classic PC (pixel) | Smooth |
|---|---|---|---|
| Resolution | 640×360 (16:9) | 320×200 | 1280×720 (16:9) |
| Palette | Resurrect 64 (64 colours) | Sweetie 16 (16 colours) | Resurrect 64 |
| Scaling | whole-number | whole-number, or CRT 4:3 | smooth, any size |
| Fonts | 8×8 and 16×16 bitmap | 8×8 and 16×16 bitmap | Go Mono Bold, 20 and 40 |
| Sound | `Chip4`: 4 channels (2 pulse, triangle, noise) | `Speaker`: 1 square-wave voice | `Chip4` |

A game can also add its own `Preset` in `internal/config/preset.go`. The game
runs at 60 ticks per second, and all game logic counts in ticks.

A game with two looks names the second as `Second` in `project.go`, beside
`Project`, which players start on. The two must share a palette and sound,
one pixel and one smooth, with the smooth screen a whole multiple of the
pixel one: Smooth and Modern (×2) are such a pair. The look is chosen in
Options and saved.

## Starting a new game

### With a coding agent

1. On GitHub, press **Use this template** to make `comprofix/<game>`, and clone
   it.
2. Start your coding agent in the folder. With no `DESIGN.md` or `PLAN.md`, it
   follows the **New game setup** section of `AGENTS.md`: it asks about the
   game, including pixel art, smooth or both, and whether to set up CI/CD, then
   (once you agree) runs `tools/newgame`, sets the preset, sets up CI/CD if
   wanted, writes `DESIGN.md` and `PLAN.md`, and updates `AGENTS.md`. It stops
   there, before any game code. Keep personal instructions in the local,
   untracked configuration supported by your chosen agent.

### By hand

1. Make and clone the repo as above.
2. Run:

   ```sh
   go run ./tools/newgame -module github.com/comprofix/<game> -name <game> -title "<Game Title>" -style pixel
   ```

   It changes the module path and every import, renames `cmd/gameskeleton` to
   `cmd/<game>`, sets the title and the preset for the style (`-style pixel`
   gives Modern, `-style smooth` gives Smooth, `-style both` gives Smooth
   with Modern as the second look), removes the demo (add `-keep-demo` to
   keep it, for pixel art only), writes a new README, removes itself, and
   builds and tests the result.
3. For pixel art with another preset, set it in
   `internal/config/project.go`.
4. Wire in your play scene, sounds and sprite sheets through `game.Content`,
   and, if you like, a demo to run behind the title (`NewDemo`) and your own
   title and high score tunes (`TitleTune`, `ScoresTune`),
   in a file such as `internal/game/<game>.go`. Until then, Play is greyed out
   in the menu. With two looks, set `NativePlay` once the play scene is drawn
   for both.
5. Update §1 of `AGENTS.md` with the game's description and fixed decisions.
   The generator has already replaced §2 with the "Working documents" section
   and removed skeleton-only setup and maintenance guidance. Keep
   `AGENTS.md` tracked; keep personal instructions in your agent's local,
   untracked configuration.
6. If you want release builds, set up CI/CD as in [Releases](#releases).

## Commands

```sh
go run ./cmd/gameskeleton                 # run the game
go run -tags debug ./cmd/gameskeleton     # with the debug overlay and Tab test pattern
go run ./cmd/gameskeleton -seed 42        # with a fixed random seed
go run ./cmd/gameskeleton -level 5        # start games on level 5, for a game with levels (no scores)
go generate ./assets                      # rebuild sprite sheets from assets/sprites/*.txt
go run ./tools/spritegen -in assets/sprites -out assets/gen -preview <dir>   # preview sheets
go vet ./...
go test ./...
go build -o bin/ ./cmd/gameskeleton
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/gameskeleton.exe ./cmd/gameskeleton
```

Linux builds need cgo, so build them on Linux or in WSL.

## Controls

| Action | Keyboard | Gamepad (standard layout) | Mouse |
|---|---|---|---|
| Move | Arrow keys, WASD | D-pad, left stick | in menus: point, or the wheel |
| Confirm | Enter, Space | A | click |
| Back | Esc, Backspace | B | right click |
| Pause | Esc, P | Start | right click |
| Restart | Ctrl+R | — | — |
| Main menu | Ctrl+M | Back/Select | — |
| Sound on/off | Ctrl+S | — | — |
| Scale mode | F9 | — | — |
| Quit | F10 | — | — |
| Fullscreen | F11, Alt+Enter | — | — |

F9, F10, F11 and Alt+Enter are kept for the window and never reach the game.
In menus a click on a slider's bar sets it, and dragging moves it. When
entering initials the wheel turns the letter and a click moves on, and the
wheel turns the high score tables.

## Saved files

Both are written next to the game (or in the working folder under `go run`),
atomically, so a crash can't leave half a file.

- `settings.json`: window size, fullscreen, scale mode, sound, music and
  effects switches, master, music and effects volumes, reduced motion, and
  in a game with two looks the look.
- `scores.json`: a top-10 table for each skill, with 3-letter initials.

A missing or broken file means defaults, never a crash, and every value is
checked when loaded.

## Testing

- Game logic never touches graphics or sound, so it runs in plain unit tests.
- `internal/scene/scenetest` runs scenes headless from a seed and a script of
  inputs. `scenetest.Golden` compares a result with `testdata/*.golden`;
  after an intended change, rewrite them with `go test ./<package> -update`.
- `go run -tags debug` adds an overlay with ticks and frames per second, the
  tick count and the scene, and lets scenes draw collision boxes.

## Releases

A game can have its releases built by GitHub Actions. The templates are in
`tools/release/`, and `AGENTS.md` §5 gives the steps to set them up and to
release a version. Once set up, pushing a tag `vX.Y.Z` tests the game on
Linux and Windows, builds a Windows zip and a Linux tarball, and publishes
them as a GitHub release with checksums and notes from `CHANGELOG.md`.
Nothing runs on an ordinary push. The release fails until the placeholder
program icon is replaced with the game's own.

## Keeping games and the skeleton in step

The skeleton is tagged by version (`v0.1.0`, …), and each new game's README
records the version it came from. A fix made in a game that would help every
game is copied back into the skeleton by hand and released as a new tag.
Existing games are not updated automatically.

## Personal documents

`DESIGN.md` and `PLAN.md` are personal working documents. `.gitignore` keeps
them out of git, so back them up somewhere else. `AGENTS.md` is tracked, so everyone working on the game gets the same shared
guidance. `CLAUDE.md` is a compatibility hand-off for Claude Code. Keep
personal instructions in the local, untracked configuration supported by
your chosen coding agent.

## License

MIT. See [LICENSE](LICENSE).
