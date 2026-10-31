package session

import (
	"testing"

	"github.com/comprofix/gameskeleton/internal/config"
)

func TestScoreStaysInRange(t *testing.T) {
	s := New(config.Skills[0], 3)
	s.AddScore(50)
	s.AddScore(-80)
	if s.Score != 0 {
		t.Errorf("score %d, want 0", s.Score)
	}
	s.AddScore(config.MaxScore + 5)
	if s.Score != config.MaxScore {
		t.Errorf("score %d, want %d", s.Score, config.MaxScore)
	}
}

func TestLoseLife(t *testing.T) {
	s := New(config.Skills[0], 2)
	if s.LoseLife() {
		t.Error("game over with a life left")
	}
	if !s.LoseLife() || s.Lives != 0 {
		t.Error("not game over after the last life")
	}
	if !s.LoseLife() || s.Lives != 0 {
		t.Error("lives went below zero")
	}
}
