package demo

import "testing"

func TestSounds(t *testing.T) {
	if err := Sounds.Check(); err != nil {
		t.Error(err)
	}
}
