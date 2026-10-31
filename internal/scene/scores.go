package scene

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/gfx"
	"github.com/comprofix/gameskeleton/internal/input"
	"github.com/comprofix/gameskeleton/internal/score"
	"github.com/comprofix/gameskeleton/internal/ui"
)

// Tally timing.
const (
	TallyLineTicks  = config.TPS     // to count up one line
	TallyPauseTicks = config.TPS / 3 // between lines
	TallySoundTicks = 3              // a tick sound every this many ticks while counting
)

// TallyLine is one line of a tally, e.g. "Dots 14 x 10" for 140 points.
type TallyLine struct {
	Label  string
	Points int
}

// Tally counts up the points of each line in turn, with a running total.
// Confirm finishes the count, then continues to next().
type Tally struct {
	heading string
	lines   []TallyLine
	next    func() Scene
	shown   []int // points counted so far on each line
	line    int   // the line being counted; len(lines) once done
	pause   int
	ticks   int64
}

// NewTally returns a tally of lines under heading.
func NewTally(heading string, lines []TallyLine, next func() Scene) *Tally {
	return &Tally{heading: heading, lines: lines, next: next, shown: make([]int, len(lines))}
}

// Done reports whether every line has been counted.
func (t *Tally) Done() bool { return t.line >= len(t.lines) }

// Total is the points counted so far.
func (t *Tally) Total() int {
	sum := 0
	for _, n := range t.shown {
		sum += n
	}
	return sum
}

// Update implements Scene.
func (t *Tally) Update(ctx *Context) error {
	t.ticks++
	if ctx.Input.JustPressed(input.Confirm) || ctx.Input.Clicked() {
		if t.Done() {
			ctx.Audio.PlayEffect(MenuConfirm)
			ctx.Switch(t.next())
			return nil
		}
		for i, l := range t.lines {
			t.shown[i] = l.Points
		}
		t.line = len(t.lines)
		return nil
	}
	if t.Done() {
		return nil
	}
	if t.pause > 0 {
		t.pause--
		return nil
	}
	l := t.lines[t.line]
	step := max(1, (l.Points+TallyLineTicks-1)/TallyLineTicks)
	t.shown[t.line] = min(t.shown[t.line]+step, l.Points)
	if t.ticks%TallySoundTicks == 0 && l.Points > 0 {
		ctx.Audio.PlayEffect(TallyTick)
	}
	if t.shown[t.line] == l.Points {
		t.line++
		t.pause = TallyPauseTicks
	}
	return nil
}

// Draw implements Scene.
func (t *Tally) Draw(ctx *Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	h := ctx.Preset.ScreenHeight
	dst.Fill(gfx.RGBA(sc.Backdrop))
	y := f.Large.LineHeight()
	ui.TextCentred(dst, f.Large, t.heading, y, sc.Title, sc.Shadow)

	// One row per line, label on the left and points on the right, with
	// the total below a gap.
	const pointsCols = 7
	width := 0
	for _, l := range t.lines {
		width = max(width, len(l.Label))
	}
	width = max(width, len("Total")) + 2 + pointsCols
	row := func(i int, label string, points int, c int) {
		s := label + strings.Repeat(" ", width-len(label)-pointsCols) + fmt.Sprintf("%*d", pointsCols, points)
		ry := y + 2*f.Large.LineHeight() + i*(f.Small.LineHeight()+ui.LineGap)
		f.Small.DrawShadowed(dst, s, f.Small.CentreX(dst, s), ry, c, sc.Shadow)
	}
	for i, l := range t.lines {
		if i > t.line {
			break
		}
		c := sc.Text
		if i == t.line {
			c = sc.Highlight
		}
		row(i, l.Label, t.shown[i], c)
	}
	row(len(t.lines)+1, "Total", t.Total(), sc.Title)
	if t.Done() && ui.Blink(t.ticks, BlinkTicks) {
		ui.TextCentred(dst, f.Small, "PRESS ENTER", h-3*f.Small.LineHeight(), sc.Text, sc.Shadow)
	}
}

// AfterGame is where a finished game goes once its score is final: initials
// entry if the score makes the table, otherwise the table itself. A game
// started for testing (Context.Testing) is not entered.
func AfterGame(ctx *Context, skill config.Skill, points int) Scene {
	if !ctx.Testing() && ctx.Scores.Qualifies(skill.ID, points) {
		return NewHighScoreEntry(skill, points)
	}
	return NewHighScoreTable(skill, -1)
}

// HighScoreEntry takes the player's initials for a new high score: Up and
// Down change the letter, Right or Confirm moves to the next, Left or Back
// to the one before.
type HighScoreEntry struct {
	skill   config.Skill
	points  int
	letters [config.InitialsLen]int // indexes into score.Letters
	slot    int
	ticks   int64
}

// NewHighScoreEntry asks for the initials for points at skill.
func NewHighScoreEntry(skill config.Skill, points int) *HighScoreEntry {
	return &HighScoreEntry{skill: skill, points: points}
}

// Initials are the letters chosen so far.
func (e *HighScoreEntry) Initials() string {
	var b strings.Builder
	for _, l := range e.letters {
		b.WriteByte(score.Letters[l])
	}
	return b.String()
}

// Update implements Scene.
func (e *HighScoreEntry) Update(ctx *Context) error {
	e.ticks++
	playScoresTune(ctx)
	in := ctx.Input
	n := len(score.Letters)
	switch {
	case in.Repeat(input.Up), in.Wheel() > 0:
		e.letters[e.slot] = (e.letters[e.slot] + 1) % n
		ctx.Audio.PlayEffect(MenuMove)
	case in.Repeat(input.Down), in.Wheel() < 0:
		e.letters[e.slot] = (e.letters[e.slot] + n - 1) % n
		ctx.Audio.PlayEffect(MenuMove)
	case in.JustPressed(input.Left), in.JustPressed(input.Back):
		if e.slot > 0 {
			e.slot--
			ctx.Audio.PlayEffect(MenuMove)
		}
	case in.JustPressed(input.Right), in.JustPressed(input.Confirm), in.Clicked():
		ctx.Audio.PlayEffect(MenuConfirm)
		if e.slot < config.InitialsLen-1 {
			e.slot++
			return nil
		}
		place := ctx.Scores.Add(e.skill.ID, e.Initials(), e.points)
		ctx.ScoresChanged()
		ctx.Switch(NewHighScoreTable(e.skill, place))
	}
	return nil
}

// Draw implements Scene.
func (e *HighScoreEntry) Draw(ctx *Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	h := ctx.Preset.ScreenHeight
	dst.Fill(gfx.RGBA(sc.Backdrop))
	y := f.Large.LineHeight()
	ui.TextCentred(dst, f.Large, "NEW HIGH SCORE", y, sc.Title, sc.Shadow)
	y += 2 * f.Large.LineHeight()
	ui.TextCentred(dst, f.Small, fmt.Sprintf("%s  %d", e.skill.Name, e.points), y, sc.Text, sc.Shadow)
	y += 3 * f.Small.LineHeight()
	ui.TextCentred(dst, f.Small, "Enter your initials", y, sc.Text, sc.Shadow)
	y += 2 * f.Small.LineHeight()

	// The letters in the large font, spaced a cell apart, with the current
	// one highlighted and underlined.
	cell := 2 * f.Large.CellWidth()
	x0 := (ctx.Preset.ScreenWidth - (config.InitialsLen*cell - f.Large.CellWidth())) / 2
	for i, l := range e.letters {
		c := sc.Text
		x := x0 + i*cell
		if i == e.slot {
			c = sc.Highlight
			if ui.Blink(e.ticks, BlinkTicks) {
				f.Large.DrawShadowed(dst, "_", x, y+f.Large.LineHeight()/4, c, sc.Shadow)
			}
		}
		f.Large.DrawShadowed(dst, string(score.Letters[l]), x, y, c, sc.Shadow)
	}
	ui.TextCentred(dst, f.Small, "Up/Down: letter   Enter: next", h-2*f.Small.LineHeight(), sc.Dim, sc.Shadow)
}

// HighScoreTable shows one skill's table. Left and Right show the other
// skills; Confirm or Back returns to the main menu.
type HighScoreTable struct {
	skill int // index into config.Skills
	mark  int // the place to highlight, or -1
	ticks int64
}

// NewHighScoreTable shows skill's table with place highlighted (-1 for
// none).
func NewHighScoreTable(skill config.Skill, place int) *HighScoreTable {
	t := &HighScoreTable{mark: place}
	for i, sk := range config.Skills {
		if sk.ID == skill.ID {
			t.skill = i
		}
	}
	return t
}

// Update implements Scene.
func (t *HighScoreTable) Update(ctx *Context) error {
	t.ticks++
	playScoresTune(ctx)
	in := ctx.Input
	n := len(config.Skills)
	switch {
	case in.JustPressed(input.Left), in.Wheel() > 0:
		t.skill, t.mark = (t.skill+n-1)%n, -1
		ctx.Audio.PlayEffect(MenuMove)
	case in.JustPressed(input.Right), in.Wheel() < 0:
		t.skill, t.mark = (t.skill+1)%n, -1
		ctx.Audio.PlayEffect(MenuMove)
	case in.JustPressed(input.Confirm), in.JustPressed(input.Back), in.Clicked():
		ctx.Audio.PlayEffect(MenuConfirm)
		ctx.Switch(NewMainMenu())
	}
	return nil
}

// Draw implements Scene.
func (t *HighScoreTable) Draw(ctx *Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	h := ctx.Preset.ScreenHeight
	dst.Fill(gfx.RGBA(sc.Backdrop))
	y := f.Large.LineHeight()
	ui.TextCentred(dst, f.Large, "HIGH SCORES", y, sc.Title, sc.Shadow)
	y += f.Large.LineHeight() + ui.LineGap + menuPad(f)
	skill := config.Skills[t.skill]
	name := skill.Name
	if len(config.Skills) > 1 {
		name = "< " + name + " >"
	}
	ui.TextCentred(dst, f.Small, name, y, sc.Highlight, sc.Shadow)
	y += f.Small.LineHeight() + 2*ui.LineGap

	es := ctx.Scores.Table(skill.ID)
	for i := range config.TableSize {
		s := fmt.Sprintf("%2d.  ---  %6s", i+1, "-")
		c := sc.Dim
		if i < len(es) {
			s = fmt.Sprintf("%2d.  %s  %6d", i+1, es[i].Initials, es[i].Score)
			c = sc.Text
		}
		if i == t.mark {
			c = sc.Highlight
			if !ui.Blink(t.ticks, BlinkTicks) {
				c = sc.Title
			}
		}
		f.Small.DrawShadowed(dst, s, f.Small.CentreX(dst, s), y, c, sc.Shadow)
		y += f.Small.LineHeight() + ui.LineGap
	}
	ui.TextCentred(dst, f.Small, "Left/Right: skill   Enter: menu", h-2*f.Small.LineHeight(), sc.Dim, sc.Shadow)
}
