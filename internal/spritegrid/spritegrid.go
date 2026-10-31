// Package spritegrid parses text-grid sprite sources and packs them into a
// sprite sheet. Format:
//
//	; comment
//	colours k=Ink g=Moss y=Gold          optional colour key, before any sprite
//	sprite NAME WIDTH HEIGHT
//	<HEIGHT rows of WIDTH characters>    frame 0
//	<blank line>
//	<HEIGHT rows>                        frame 1, and so on
//	recolour NEW from SRC g=y            a copy of SRC (from the same file)
//	                                     with colours swapped
//
// '.' is a see-through pixel. Without a colours line, every other character
// is a hex digit 0–F naming a palette index, which suits palettes of up to
// 16 colours. With one, every other character must be a key character, and
// each key entry names a colour of the palette. recolour swaps use the same
// characters as the pixels.
package spritegrid

import (
	"bufio"
	"fmt"
	"image"
	"io"
	"strconv"
	"strings"

	"github.com/comprofix/gameskeleton/internal/gfx/palettes"
)

// Transparent marks an empty pixel.
const Transparent = -1

// MaxSize is the largest sprite width or height.
const MaxSize = 1024

// Sprite is one named sprite with one or more frames of equal size.
type Sprite struct {
	Name   string
	W, H   int
	Frames [][]int // each W*H palette indices, row-major
}

// Entry locates a sprite's frames in a packed sheet. Frames run left to
// right.
type Entry struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	W      int `json:"w"`
	H      int `json:"h"`
	Frames int `json:"frames"`
}

// parser holds the state while reading one file.
type parser struct {
	pal     palettes.Palette
	key     map[rune]int // nil until a colours line is read
	sprites []Sprite
	cur     *Sprite
	rows    []string
	lineNo  int
}

// recolour is a sprite made from another by swapping palette indices.
type recolour struct {
	line       int
	name, from string
	swap       map[int]int
}

// Parse reads all sprites from r, resolving colours against pal.
func Parse(r io.Reader, pal palettes.Palette) ([]Sprite, error) {
	p := &parser{pal: pal}
	var recolours []recolour
	names := map[string]int{} // name → line it was defined on

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		p.lineNo++
		line := strings.TrimRight(sc.Text(), " \t\r")
		switch {
		case strings.HasPrefix(line, ";"):
			continue
		case line == "colours" || strings.HasPrefix(line, "colours "):
			if p.key != nil {
				return nil, p.errorf("only one colours line is allowed")
			}
			if len(p.sprites) > 0 || len(recolours) > 0 {
				return nil, p.errorf("the colours line must come before any sprite")
			}
			if err := p.parseKey(line); err != nil {
				return nil, err
			}
		case strings.HasPrefix(line, "recolour "):
			if err := p.flush(); err != nil {
				return nil, err
			}
			rc, err := p.parseRecolour(line)
			if err != nil {
				return nil, err
			}
			if prev, dup := names[rc.name]; dup {
				return nil, p.errorf("sprite %q is already defined on line %d", rc.name, prev)
			}
			names[rc.name] = p.lineNo
			recolours = append(recolours, rc)
			p.cur = nil
		case strings.HasPrefix(line, "sprite "):
			if err := p.flush(); err != nil {
				return nil, err
			}
			var s Sprite
			var extra string
			if n, _ := fmt.Sscanf(line, "sprite %s %d %d %s", &s.Name, &s.W, &s.H, &extra); n != 3 {
				return nil, p.errorf("want \"sprite NAME WIDTH HEIGHT\", got %q", line)
			}
			if s.W <= 0 || s.H <= 0 || s.W > MaxSize || s.H > MaxSize {
				return nil, p.errorf("sprite %q is %d×%d; sizes must be 1–%d", s.Name, s.W, s.H, MaxSize)
			}
			if prev, dup := names[s.Name]; dup {
				return nil, p.errorf("sprite %q is already defined on line %d", s.Name, prev)
			}
			names[s.Name] = p.lineNo
			p.sprites = append(p.sprites, s)
			p.cur = &p.sprites[len(p.sprites)-1]
		case line == "":
			if err := p.flush(); err != nil {
				return nil, err
			}
		default:
			if p.cur == nil {
				return nil, p.errorf("pixels before any sprite header")
			}
			if n := len([]rune(line)); n != p.cur.W {
				return nil, p.errorf("sprite %q row is %d wide, want %d", p.cur.Name, n, p.cur.W)
			}
			p.rows = append(p.rows, line)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if err := p.flush(); err != nil {
		return nil, err
	}
	for _, s := range p.sprites {
		if len(s.Frames) == 0 {
			return nil, fmt.Errorf("sprite %q has no frames", s.Name)
		}
	}
	for _, rc := range recolours {
		s, err := rc.apply(p.sprites)
		if err != nil {
			return nil, err
		}
		p.sprites = append(p.sprites, s)
	}
	return p.sprites, nil
}

func (p *parser) errorf(format string, args ...any) error {
	return fmt.Errorf("line %d: %s", p.lineNo, fmt.Sprintf(format, args...))
}

// parseKey reads "colours a=Name b=Name ...".
func (p *parser) parseKey(line string) error {
	f := strings.Fields(line)
	if f[0] != "colours" || len(f) < 2 {
		return p.errorf("want \"colours c=Name ...\", got %q", line)
	}
	p.key = map[rune]int{}
	for _, entry := range f[1:] {
		ch, name, ok := strings.Cut(entry, "=")
		r := []rune(ch)
		if !ok || len(r) != 1 || name == "" {
			return p.errorf("bad colour key entry %q: want c=Name", entry)
		}
		if r[0] == '.' || r[0] == ';' {
			return p.errorf("%q cannot be a key character", r[0])
		}
		if _, dup := p.key[r[0]]; dup {
			return p.errorf("key character %q is used twice", r[0])
		}
		idx, found := p.pal.Lookup(name)
		if !found {
			return p.errorf("palette %s has no colour named %q", p.pal.Name, name)
		}
		p.key[r[0]] = idx
	}
	return nil
}

// colour turns a pixel or swap character into a palette index.
func (p *parser) colour(ch rune) (int, error) {
	if p.key != nil {
		idx, ok := p.key[ch]
		if !ok {
			return 0, fmt.Errorf("%q is not in the colours key", ch)
		}
		return idx, nil
	}
	v, err := strconv.ParseInt(string(ch), 16, 8)
	if err != nil {
		return 0, fmt.Errorf("bad pixel %q: want a hex digit, or add a colours line", ch)
	}
	if int(v) >= p.pal.Len() {
		return 0, fmt.Errorf("colour %X is outside palette %s (%d colours)", v, p.pal.Name, p.pal.Len())
	}
	return int(v), nil
}

// flush ends the current frame, if any.
func (p *parser) flush() error {
	if p.cur == nil || len(p.rows) == 0 {
		return nil
	}
	if len(p.rows) != p.cur.H {
		return p.errorf("sprite %q frame %d has %d rows, want %d", p.cur.Name, len(p.cur.Frames), len(p.rows), p.cur.H)
	}
	frame := make([]int, 0, p.cur.W*p.cur.H)
	for _, row := range p.rows {
		for _, ch := range row {
			if ch == '.' {
				frame = append(frame, Transparent)
				continue
			}
			idx, err := p.colour(ch)
			if err != nil {
				return p.errorf("sprite %q: %v", p.cur.Name, err)
			}
			frame = append(frame, idx)
		}
	}
	p.cur.Frames = append(p.cur.Frames, frame)
	p.rows = p.rows[:0]
	return nil
}

// parseRecolour reads "recolour NEW from SRC a=b ...".
func (p *parser) parseRecolour(line string) (recolour, error) {
	f := strings.Fields(line)
	if len(f) < 5 || f[2] != "from" {
		return recolour{}, p.errorf("want \"recolour NEW from SRC a=b ...\", got %q", line)
	}
	rc := recolour{line: p.lineNo, name: f[1], from: f[3], swap: map[int]int{}}
	for _, pair := range f[4:] {
		a, b, ok := strings.Cut(pair, "=")
		ra, rb := []rune(a), []rune(b)
		if !ok || len(ra) != 1 || len(rb) != 1 {
			return recolour{}, p.errorf("bad colour swap %q: want a=b", pair)
		}
		from, err := p.colour(ra[0])
		if err != nil {
			return recolour{}, p.errorf("colour swap %q: %v", pair, err)
		}
		to, err := p.colour(rb[0])
		if err != nil {
			return recolour{}, p.errorf("colour swap %q: %v", pair, err)
		}
		rc.swap[from] = to
	}
	return rc, nil
}

// apply builds the recoloured sprite from its source.
func (rc recolour) apply(sprites []Sprite) (Sprite, error) {
	for _, s := range sprites {
		if s.Name != rc.from {
			continue
		}
		out := Sprite{Name: rc.name, W: s.W, H: s.H}
		for _, fr := range s.Frames {
			nf := make([]int, len(fr))
			for i, v := range fr {
				if to, ok := rc.swap[v]; ok {
					v = to
				}
				nf[i] = v
			}
			out.Frames = append(out.Frames, nf)
		}
		return out, nil
	}
	return Sprite{}, fmt.Errorf("line %d: recolour %q: no sprite %q in this file", rc.line, rc.name, rc.from)
}

// Pack lays sprites out top to bottom, frames left to right, and returns
// the sheet image in pal's colours and the location of each sprite.
func Pack(sprites []Sprite, pal palettes.Palette) (*image.RGBA, map[string]Entry, error) {
	w, h := 1, 1
	for _, s := range sprites {
		w = max(w, s.W*len(s.Frames))
		h += s.H
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	index := make(map[string]Entry, len(sprites))
	y := 0
	for _, s := range sprites {
		if _, dup := index[s.Name]; dup {
			return nil, nil, fmt.Errorf("duplicate sprite name %q", s.Name)
		}
		for f, frame := range s.Frames {
			for i, c := range frame {
				if c == Transparent {
					continue
				}
				if c < 0 || c >= pal.Len() {
					return nil, nil, fmt.Errorf("sprite %q uses colour %d, outside palette %s", s.Name, c, pal.Name)
				}
				img.SetRGBA(f*s.W+i%s.W, y+i/s.W, pal.Colors[c])
			}
		}
		index[s.Name] = Entry{X: 0, Y: y, W: s.W, H: s.H, Frames: len(s.Frames)}
		y += s.H
	}
	return img, index, nil
}
