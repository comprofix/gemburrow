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

func TestLevelForRepeats(t *testing.T) {
	if &LevelFor(1)[0] != &LevelFor(len(Levels) + 1)[0] {
		t.Error("levels do not repeat after the last")
	}
}
