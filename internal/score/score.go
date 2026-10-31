// Package score keeps the high score tables: the best config.TableSize
// scores for each skill, with initials, saved between runs.
package score

import (
	"encoding/json"
	"os"
	"slices"
	"strings"

	"github.com/comprofix/gameskeleton/internal/config"
)

// Entry is one line of a table.
type Entry struct {
	Initials string `json:"initials"`
	Score    int    `json:"score"`
}

// Tables holds a table for each skill, keyed by Skill.ID, best first.
type Tables struct {
	BySkill map[string][]Entry `json:"tables"`
}

// New returns empty tables.
func New() *Tables { return &Tables{BySkill: map[string][]Entry{}} }

// Load reads the tables at path. A missing or broken file gives empty
// tables, and every entry is checked: tables for unknown skills are
// dropped, initials are cleaned up, scores are clamped, and each table is
// sorted and cut to size. So it never fails.
func Load(path string, skills []config.Skill) *Tables {
	t := New()
	data, err := os.ReadFile(path)
	if err != nil {
		return t
	}
	var f Tables
	if json.Unmarshal(data, &f) != nil {
		return t
	}
	for _, sk := range skills {
		var es []Entry
		for _, e := range f.BySkill[sk.ID] {
			if e.Score <= 0 {
				continue
			}
			es = append(es, Entry{Initials: CleanInitials(e.Initials), Score: min(e.Score, config.MaxScore)})
		}
		slices.SortStableFunc(es, func(a, b Entry) int { return b.Score - a.Score })
		if len(es) > 0 {
			t.BySkill[sk.ID] = es[:min(len(es), config.TableSize)]
		}
	}
	return t
}

// Save writes the tables to path.
func (t *Tables) Save(path string) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteFile(path, append(data, '\n'))
}

// Table is a skill's table, best first.
func (t *Tables) Table(skill string) []Entry { return t.BySkill[skill] }

// Qualifies reports whether score would make a skill's table.
func (t *Tables) Qualifies(skill string, score int) bool {
	es := t.BySkill[skill]
	return score > 0 && (len(es) < config.TableSize || score > es[len(es)-1].Score)
}

// Add puts a score in a skill's table and returns its place, 0 for the
// top, or -1 if it didn't make the table. A score equal to one already
// there goes below it: the earlier score keeps its place.
func (t *Tables) Add(skill, initials string, score int) int {
	if !t.Qualifies(skill, score) {
		return -1
	}
	score = min(score, config.MaxScore)
	es := t.BySkill[skill]
	i := 0
	for i < len(es) && es[i].Score >= score {
		i++
	}
	es = slices.Insert(es, i, Entry{Initials: CleanInitials(initials), Score: score})
	t.BySkill[skill] = es[:min(len(es), config.TableSize)]
	return i
}

// Letters are the characters initials may use.
const Letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// CleanInitials makes s valid initials: upper case, only Letters, padded
// with "A" or cut to config.InitialsLen.
func CleanInitials(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if b.Len() == config.InitialsLen {
			break
		}
		if strings.ContainsRune(Letters, r) {
			b.WriteRune(r)
		}
	}
	for b.Len() < config.InitialsLen {
		b.WriteByte('A')
	}
	return b.String()
}
