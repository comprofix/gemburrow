package score

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/comprofix/gameskeleton/internal/config"
)

var skills = []config.Skill{{ID: "easy"}, {ID: "hard"}}

func TestAddKeepsOrderAndSize(t *testing.T) {
	tb := New()
	for i := range config.TableSize {
		if got := tb.Add("easy", "abc", 100+i*10); got != 0 {
			t.Fatalf("score %d went to place %d, want 0", 100+i*10, got)
		}
	}
	if tb.Qualifies("easy", 100) {
		t.Error("a score equal to the lowest of a full table qualifies")
	}
	if got := tb.Add("easy", "low", 50); got != -1 {
		t.Errorf("a score below a full table went to place %d", got)
	}
	if got := tb.Add("easy", "tie", 150); got != 5 {
		t.Errorf("a tie with 150 went to place %d, want 5 (below the earlier 150)", got)
	}
	es := tb.Table("easy")
	if len(es) != config.TableSize {
		t.Fatalf("%d entries, want %d", len(es), config.TableSize)
	}
	if es[0].Score != 190 || es[len(es)-1].Score != 110 || es[0].Initials != "ABC" {
		t.Errorf("table %v", es)
	}
	if len(tb.Table("hard")) != 0 {
		t.Error("adding to easy changed hard")
	}
}

func TestZeroDoesNotQualify(t *testing.T) {
	if New().Qualifies("easy", 0) {
		t.Error("a score of 0 qualifies")
	}
}

func TestCleanInitials(t *testing.T) {
	for in, want := range map[string]string{
		"abc": "ABC", "ABCD": "ABC", "a": "AAA", "": "AAA", "a1b2c": "ABC", "é z": "ZAA",
	} {
		if got := CleanInitials(in); got != want {
			t.Errorf("CleanInitials(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), config.ScoresFile)
	tb := New()
	tb.Add("easy", "ABC", 300)
	tb.Add("easy", "DEF", 200)
	tb.Add("hard", "GHI", 50)
	if err := tb.Save(path); err != nil {
		t.Fatal(err)
	}
	got := Load(path, skills)
	if !reflect.DeepEqual(got, tb) {
		t.Errorf("loaded %+v, want %+v", got, tb)
	}
}

func TestLoadMissingOrBroken(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.json")
	os.WriteFile(broken, []byte("{not json"), 0o644)
	for _, path := range []string{filepath.Join(dir, "missing.json"), broken} {
		if got := Load(path, skills); len(got.BySkill) != 0 {
			t.Errorf("%s: got %v, want empty tables", filepath.Base(path), got.BySkill)
		}
	}
}

func TestLoadChecksEveryEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), config.ScoresFile)
	os.WriteFile(path, []byte(`{"tables": {
		"easy": [
			{"initials": "zz", "score": 5},
			{"initials": "toolong", "score": 99999999},
			{"initials": "neg", "score": -4},
			{"initials": "abc", "score": 7},
			{"initials": "d", "score": 6}, {"initials": "e", "score": 6}, {"initials": "f", "score": 6},
			{"initials": "g", "score": 6}, {"initials": "h", "score": 6}, {"initials": "i", "score": 6},
			{"initials": "j", "score": 6}, {"initials": "k", "score": 6}
		],
		"nightmare": [{"initials": "XYZ", "score": 1}]
	}}`), 0o644)
	got := Load(path, skills)
	if _, ok := got.BySkill["nightmare"]; ok {
		t.Error("kept a table for an unknown skill")
	}
	es := got.Table("easy")
	if len(es) != config.TableSize {
		t.Fatalf("%d entries, want %d", len(es), config.TableSize)
	}
	want0 := Entry{Initials: "TOO", Score: config.MaxScore}
	if es[0] != want0 || es[1].Initials != "ABC" || es[2].Initials != "DAA" {
		t.Errorf("first entries %v, want %v then ABC, DAA (sorted, ties in file order)", es[:3], want0)
	}
	for _, e := range es {
		if e.Score <= 0 || e.Initials == "ZZA" {
			t.Errorf("kept %v, which should be gone (negative, or cut by size)", e)
		}
	}
}
