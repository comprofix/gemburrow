package scene

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/input"
)

func TestTallyCountsUpThenContinues(t *testing.T) {
	var next *MainMenu
	tally := NewTally("DONE", []TallyLine{{"A", 120}, {"B", 30}, {"C", 0}}, func() Scene {
		next = NewMainMenu()
		return next
	})
	h := newHarness(t, tally)
	if tally.Done() {
		t.Fatal("done before counting")
	}
	h.ticks(TallyLineTicks / 2)
	if got := tally.Total(); got <= 0 || got >= 120 {
		t.Errorf("half way through the first line, total %d", got)
	}
	h.ticks(3*(TallyLineTicks+TallyPauseTicks) + 1)
	if !tally.Done() || tally.Total() != 150 {
		t.Fatalf("after counting: done %v, total %d", tally.Done(), tally.Total())
	}
	h.press(ebiten.KeyEnter)
	if current[*MainMenu](t, h) != next {
		t.Error("Enter did not go to the next scene")
	}
}

func TestTallyConfirmSkipsTheCount(t *testing.T) {
	tally := NewTally("DONE", []TallyLine{{"A", 500}, {"B", 500}}, func() Scene { return NewMainMenu() })
	h := newHarness(t, tally)
	h.press(ebiten.KeyEnter)
	if !tally.Done() || tally.Total() != 1000 {
		t.Fatalf("after Enter: done %v, total %d", tally.Done(), tally.Total())
	}
	current[*Tally](t, h) // the first Enter only finishes the count
}

func TestAfterGame(t *testing.T) {
	h := newHarness(t, NewMainMenu())
	sk := config.Skills[0]
	if _, ok := AfterGame(h.ctx, sk, 0).(*HighScoreTable); !ok {
		t.Error("a score of 0 should go straight to the table")
	}
	if _, ok := AfterGame(h.ctx, sk, 10).(*HighScoreEntry); !ok {
		t.Error("a first score should ask for initials")
	}
}

func TestEnterInitials(t *testing.T) {
	sk := config.Skills[1]
	entry := NewHighScoreEntry(sk, 420)
	h := newHarness(t, entry)
	h.ctx.Scores.Add(sk.ID, "TOP", 999)
	h.press(ebiten.KeyArrowUp)
	h.press(ebiten.KeyArrowUp) // C
	h.press(ebiten.KeyEnter)
	h.press(ebiten.KeyArrowDown) // Z, wrapping
	h.press(ebiten.KeyArrowRight)
	h.press(ebiten.KeyArrowLeft) // back to the Z
	h.press(ebiten.KeyArrowDown) // Y
	h.press(ebiten.KeyEnter)
	if got := entry.Initials(); got != "CYA" {
		t.Fatalf("initials %q, want CYA", got)
	}
	h.press(ebiten.KeyEnter)
	table := current[*HighScoreTable](t, h)
	if table.mark != 1 || config.Skills[table.skill].ID != sk.ID {
		t.Errorf("table shows skill %d with place %d marked, want %q with place 1", table.skill, table.mark, sk.ID)
	}
	es := h.ctx.Scores.Table(sk.ID)
	if len(es) != 2 || es[1].Initials != "CYA" || es[1].Score != 420 {
		t.Errorf("table %v", es)
	}
	if !h.ctx.TakeScoresChanged() {
		t.Error("the new score was not reported for saving")
	}
}

func TestHighScoreTableFromTheMenu(t *testing.T) {
	h := newHarness(t, NewMainMenu())
	h.ticks(1) // builds the items
	m := current[*MainMenu](t, h)
	for m.menu.Items[m.menu.Cursor].Label != "High scores" {
		h.press(ebiten.KeyArrowDown)
	}
	h.press(ebiten.KeyEnter)
	table := current[*HighScoreTable](t, h)
	h.press(ebiten.KeyArrowRight)
	if table.skill != 1 {
		t.Errorf("Right went to skill %d, want 1", table.skill)
	}
	h.press(ebiten.KeyArrowRight)
	if table.skill != 0 {
		t.Errorf("Right from the last skill went to %d, want 0", table.skill)
	}
	h.press(ebiten.KeyEscape)
	current[*MainMenu](t, h)
}

// TestTestGamesMakeNoTable checks a game started part way in, with
// -level, is not offered the high score table.
func TestTestGamesMakeNoTable(t *testing.T) {
	h := newHarness(t, NewMainMenu())
	if _, ok := AfterGame(h.ctx, config.Skills[0], 5000).(*HighScoreEntry); !ok {
		t.Fatal("a normal game was not offered the table")
	}
	h.ctx.StartLevel = 20
	if _, ok := AfterGame(h.ctx, config.Skills[0], 5000).(*HighScoreTable); !ok {
		t.Error("a game started part way was offered the table")
	}
}

// TestScoresByMouse: the wheel turns letters and tables, and a click moves
// on, so initials and the tables need no keys.
func TestScoresByMouse(t *testing.T) {
	sk := config.Skills[0]
	entry := NewHighScoreEntry(sk, 420)
	h := newHarness(t, entry)
	wheel := func(dy float64) {
		h.in.Steps = append(h.in.Steps, input.Step{Wheel: dy, Ticks: 1}, input.Idle(1))
		h.ticks(2)
	}
	wheel(1)
	wheel(1) // C
	h.mouse(input.Click(0, 0))
	wheel(-1) // Z, wrapping
	h.mouse(input.Click(0, 0))
	h.mouse(input.Click(0, 0)) // A, the last
	if got := entry.Initials(); got != "CZA" {
		t.Fatalf("initials %q, want CZA", got)
	}
	table := current[*HighScoreTable](t, h)
	wheel(-1)
	if table.skill != 1 {
		t.Errorf("the wheel went to skill %d, want 1", table.skill)
	}
	h.mouse(input.Click(0, 0))
	current[*MainMenu](t, h)
}
