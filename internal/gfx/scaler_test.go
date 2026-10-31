package gfx

import (
	"testing"

	"github.com/comprofix/gameskeleton/internal/config"
)

func TestFitInteger(t *testing.T) {
	cases := []struct {
		name       string
		sw, sh     int
		outW, outH int
		want       Rect
	}{
		{"Modern 720p", 640, 360, 1280, 720, Rect{0, 0, 1280, 720}},
		{"Modern 1080p", 640, 360, 1920, 1080, Rect{0, 0, 1920, 1080}},
		{"Modern 1440p", 640, 360, 2560, 1440, Rect{0, 0, 2560, 1440}},
		{"Modern 4K", 640, 360, 3840, 2160, Rect{0, 0, 3840, 2160}},
		{"Modern 1920x1200 letterboxed", 640, 360, 1920, 1200, Rect{0, 60, 1920, 1080}},
		{"Modern odd window", 640, 360, 1500, 900, Rect{110, 90, 1280, 720}},
		{"Classic 1080p", 320, 200, 1920, 1080, Rect{160, 40, 1600, 1000}},
		{"Classic 1280x800", 320, 200, 1280, 800, Rect{0, 0, 1280, 800}},
		{"smaller than the screen", 640, 360, 320, 360, Rect{0, 90, 320, 180}},
	}
	for _, c := range cases {
		if got := Fit(config.ScaleInteger, c.sw, c.sh, c.outW, c.outH); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestFitSmooth(t *testing.T) {
	cases := []struct {
		name       string
		outW, outH int
		want       Rect
	}{
		{"720p", 1280, 720, Rect{0, 0, 1280, 720}},
		{"1080p", 1920, 1080, Rect{0, 0, 1920, 1080}},
		{"1366x768, a fraction", 1366, 768, Rect{0, 0, 1365.3333333333333, 768}},
		{"1920x1200 letterboxed", 1920, 1200, Rect{0, 60, 1920, 1080}},
		{"a small window", 800, 600, Rect{0, 75, 800, 450}},
	}
	for _, c := range cases {
		if got := Fit(config.ScaleSmooth, 1280, 720, c.outW, c.outH); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestFitCRT(t *testing.T) {
	got := Fit(config.ScaleCRT, 320, 200, 1920, 1080)
	want := Rect{240, 0, 1440, 1080}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	// A tall window: limited by width.
	got = Fit(config.ScaleCRT, 320, 200, 800, 1000)
	want = Rect{0, 200, 800, 600}
	if got != want {
		t.Errorf("tall window: got %+v, want %+v", got, want)
	}
}

func TestPrescaleFactor(t *testing.T) {
	// 1440×1080 from 320×200: 4.5 across and 5.4 down, so prescale by 6.
	if got := PrescaleFactor(Rect{W: 1440, H: 1080}, 320, 200); got != 6 {
		t.Errorf("got %d, want 6", got)
	}
	if got := PrescaleFactor(Rect{W: 100, H: 60}, 320, 200); got != 1 {
		t.Errorf("tiny output: got %d, want 1", got)
	}
}
