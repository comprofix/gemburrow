package audio

import (
	"time"

	eaudio "github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/comprofix/gameskeleton/internal/config"
)

// Start streams s to the sound device through Ebitengine and returns the
// player, which the caller must keep alive. Call it once.
func Start(s *Synth) (*eaudio.Player, error) {
	ctx := eaudio.NewContext(config.AudioSampleRate)
	p, err := ctx.NewPlayer(s)
	if err != nil {
		return nil, err
	}
	p.SetBufferSize(config.AudioBufferMs * time.Millisecond)
	p.Play()
	return p, nil
}
