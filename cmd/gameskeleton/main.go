// Command gameskeleton starts the game: it loads the saved files, opens the
// window and runs the scenes.
package main

import (
	"flag"
	"log"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/audio"
	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/game"
	"github.com/comprofix/gameskeleton/internal/icon"
	"github.com/comprofix/gameskeleton/internal/score"
)

func main() {
	seed := flag.Uint64("seed", 0, "random seed, for repeatable runs (0 picks one from the clock)")
	level := flag.Int("level", 1, "start new games on this level, for testing; such games keep no high scores")
	flag.Parse()
	if *seed == 0 {
		*seed = uint64(time.Now().UnixNano())
	}

	looks := config.ProjectLooks()
	if err := looks.Check(); err != nil {
		log.Fatal(err)
	}

	// The settings and scores live beside the game.
	dir := config.SaveDir()
	saved := game.Saved{
		Settings: config.LoadSettings(filepath.Join(dir, config.SettingsFile), looks),
		Scores:   score.Load(filepath.Join(dir, config.ScoresFile), config.Skills),
		Dir:      dir,
	}

	ebiten.SetWindowTitle(config.Title)
	icons, err := icon.Images()
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowIcon(icons)
	ebiten.SetWindowSize(saved.Settings.WindowW, saved.Settings.WindowH)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(saved.Settings.Fullscreen)
	ebiten.SetWindowClosingHandled(true) // to save the window size on close
	ebiten.SetTPS(config.TPS)

	// Sound is made in code. With no sound device the game
	// still runs, silently.
	synth := audio.NewSynth(config.Project.Audio) // the same in both looks
	player, err := audio.Start(synth)
	if err != nil {
		log.Printf("sound: %v", err)
		synth = nil
	}
	defer func() {
		if player != nil {
			player.Close()
		}
	}()

	g, err := game.New(*seed, saved, synth)
	if err != nil {
		log.Fatal(err)
	}
	g.StartAt(*level)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
