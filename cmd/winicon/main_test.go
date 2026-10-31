package main

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"
)

// TestMakeIco reads the file back as Windows would: the header, an entry
// for each size, and a PNG of that size at each entry's offset.
func TestMakeIco(t *testing.T) {
	ico, err := makeIco()
	if err != nil {
		t.Fatal(err)
	}
	le := binary.LittleEndian
	if le.Uint16(ico[0:]) != 0 || le.Uint16(ico[2:]) != 1 {
		t.Fatalf("header % x is not an icon's", ico[:4])
	}
	want := []int{16, 32, 48, 64, 128, 256}
	if n := int(le.Uint16(ico[4:])); n != len(want) {
		t.Fatalf("%d images, want %d", n, len(want))
	}
	for i, size := range want {
		e := ico[6+16*i:]
		side := int(e[0])
		if side == 0 {
			side = 256
		}
		if side != size || int(e[1]) != int(e[0]) {
			t.Errorf("entry %d is %d×%d, want %d", i, e[0], e[1], size)
		}
		n, off := le.Uint32(e[8:]), le.Uint32(e[12:])
		if int(off+n) > len(ico) {
			t.Fatalf("entry %d runs past the end of the file", i)
		}
		img, err := png.Decode(bytes.NewReader(ico[off : off+n]))
		if err != nil {
			t.Fatalf("entry %d: %v", i, err)
		}
		if b := img.Bounds(); b.Dx() != size || b.Dy() != size {
			t.Errorf("entry %d's PNG is %v, want %d×%d", i, b.Size(), size, size)
		}
	}
}
