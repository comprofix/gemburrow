package icon

import "testing"

func TestImages(t *testing.T) {
	imgs, err := Images()
	if err != nil {
		t.Fatal(err)
	}
	want := []int{16, 32, 48, 64, 128, 256}
	if len(imgs) != len(want) {
		t.Fatalf("got %d images, want %d", len(imgs), len(want))
	}
	for i, img := range imgs {
		b := img.Bounds()
		if b.Dx() != want[i] || b.Dy() != want[i] {
			t.Errorf("image %d is %v, want %d×%d", i, b.Size(), want[i], want[i])
		}
	}
	// Each enlarged size repeats its sprite's pixels in whole blocks: the
	// 48 is the 16 tripled and the 256 is the 32 times eight.
	for _, c := range []struct{ small, big, k int }{{0, 2, 3}, {1, 5, 8}} {
		small, big := imgs[c.small], imgs[c.big]
		n := small.Bounds().Dx()
		for y := range n {
			for x := range n {
				if small.At(x, y) != big.At(x*c.k+c.k-1, y*c.k+c.k-1) {
					t.Fatalf("%d px icon at (%d,%d) doesn't match the %d px one", want[c.big], x*c.k+c.k-1, y*c.k+c.k-1, n)
				}
			}
		}
	}
}
