package config

import "testing"

func TestFieldFillsThePixelLook(t *testing.T) {
	if got := FieldX*2 + FieldCols*CellSize; got != Modern.ScreenWidth {
		t.Errorf("field and margins are %d wide, screen %d", got, Modern.ScreenWidth)
	}
	if got := FieldY + FieldRows*CellSize; got != Modern.ScreenHeight {
		t.Errorf("HUD and field are %d high, screen %d", got, Modern.ScreenHeight)
	}
}

func TestDwarfSpeedDividesTheCell(t *testing.T) {
	if CellSize%DwarfSpeed != 0 {
		t.Errorf("speed %d never lands on a cell of %d", DwarfSpeed, CellSize)
	}
}

func TestGoblinSpeedDividesTheCell(t *testing.T) {
	if CellSize%BaseTuning.GoblinSpeed != 0 {
		t.Errorf("speed %d never lands on a cell of %d", BaseTuning.GoblinSpeed, CellSize)
	}
}

func TestSackFallSpeedDividesTheCell(t *testing.T) {
	if CellSize%SackFallSpeed != 0 {
		t.Errorf("speed %d never lands on a cell of %d", SackFallSpeed, CellSize)
	}
}

// TestNormalFollowsTheOriginalRamp checks the values in DESIGN §11.
func TestNormalFollowsTheOriginalRamp(t *testing.T) {
	normal := Skills[1]
	for _, c := range []struct{ level, total, atOnce, gap, recharge, extra int }{
		{1, 6, 3, 43, 63, 14},
		{2, 7, 4, 41, 66, 13},
		{7, 12, 4, 31, 81, 8},
		{8, 13, 5, 29, 84, 7},
		{10, 15, 5, 25, 90, 5},
		{50, 15, 5, 25, 90, 5},
	} {
		got := TuningFor(normal, c.level)
		if got.GoblinTotal != c.total || got.GoblinsAtOnce != c.atOnce ||
			got.GoblinSpawnTicks != originalTicks(c.gap) || got.ShotRechargeTicks != originalTicks(c.recharge) ||
			got.GoblinExtraOneIn != c.extra {
			t.Errorf("level %d: %+v", c.level, got)
		}
	}
}

func TestSkillsOrderTheDifficulty(t *testing.T) {
	if len(Skills) != 3 {
		t.Fatalf("%d skills, want Easy, Normal and Hard", len(Skills))
	}
	for level := 1; level <= RampLevels; level++ {
		e, n, h := TuningFor(Skills[0], level), TuningFor(Skills[1], level), TuningFor(Skills[2], level)
		if e.GoblinTotal >= n.GoblinTotal || n.GoblinTotal >= h.GoblinTotal {
			t.Errorf("level %d: totals %d, %d, %d", level, e.GoblinTotal, n.GoblinTotal, h.GoblinTotal)
		}
		if e.ShotRechargeTicks >= n.ShotRechargeTicks || n.ShotRechargeTicks >= h.ShotRechargeTicks {
			t.Errorf("level %d: recharges %d, %d, %d", level, e.ShotRechargeTicks, n.ShotRechargeTicks, h.ShotRechargeTicks)
		}
		if e.GoblinSpawnTicks <= n.GoblinSpawnTicks || n.GoblinSpawnTicks <= h.GoblinSpawnTicks {
			t.Errorf("level %d: entry gaps %d, %d, %d", level, e.GoblinSpawnTicks, n.GoblinSpawnTicks, h.GoblinSpawnTicks)
		}
		if e.GoblinExtraOneIn <= n.GoblinExtraOneIn || n.GoblinExtraOneIn <= h.GoblinExtraOneIn {
			t.Errorf("level %d: extra steps %d, %d, %d", level, e.GoblinExtraOneIn, n.GoblinExtraOneIn, h.GoblinExtraOneIn)
		}
		if h.TrollAfterTicks < 0 || e.TrollAfterTicks <= n.TrollAfterTicks || h.GoblinsAtOnce > h.GoblinTotal {
			t.Errorf("level %d: hard %+v", level, h)
		}
	}
}

func TestLevelForRepeats(t *testing.T) {
	if &LevelFor(1)[0] != &LevelFor(len(Levels) + 1)[0] {
		t.Error("levels do not repeat after the last")
	}
}

func TestSackSlideSpeedIsHalfWalking(t *testing.T) {
	if SackSlideSpeed*2 != DwarfSpeed || CellSize%SackSlideSpeed != 0 {
		t.Errorf("slide speed %d, dwarf speed %d", SackSlideSpeed, DwarfSpeed)
	}
}
