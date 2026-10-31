// Command winicon writes the program icon as a Windows .ico file, for a
// release build to link into the exe so that Explorer shows it:
//
//	go run ./cmd/winicon gameskeleton.ico
//
// The images come from internal/icon, the same as the window's. Each size
// is stored in the file as a PNG, which Windows has read since Vista.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/png"
	"log"
	"os"

	"github.com/comprofix/gameskeleton/internal/icon"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: winicon file.ico")
		os.Exit(2)
	}
	if icon.Placeholder {
		fmt.Fprintln(os.Stderr, "winicon: the program icon is still the skeleton's placeholder: draw the game's own, then set icon.Placeholder to false")
		os.Exit(1)
	}
	ico, err := makeIco()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(os.Args[1], ico, 0o644); err != nil {
		log.Fatal(err)
	}
}

// makeIco lays out the file: a header, an entry for each size, then the
// images.
func makeIco() ([]byte, error) {
	imgs, err := icon.Images()
	if err != nil {
		return nil, err
	}
	var pngs [][]byte
	for _, img := range imgs {
		var b bytes.Buffer
		if err := png.Encode(&b, img); err != nil {
			return nil, err
		}
		pngs = append(pngs, b.Bytes())
	}

	le := binary.LittleEndian
	var out []byte
	out = le.AppendUint16(out, 0) // reserved
	out = le.AppendUint16(out, 1) // an icon, not a cursor
	out = le.AppendUint16(out, uint16(len(imgs)))

	offset := 6 + 16*len(imgs)
	for i, img := range imgs {
		side := byte(img.Bounds().Dx()) // 256 is written as 0
		out = append(out, side, side, 0, 0)
		out = le.AppendUint16(out, 1)  // colour planes
		out = le.AppendUint16(out, 32) // bits a pixel
		out = le.AppendUint32(out, uint32(len(pngs[i])))
		out = le.AppendUint32(out, uint32(offset))
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		out = append(out, p...)
	}
	return out, nil
}
