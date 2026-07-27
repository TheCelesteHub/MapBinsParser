package mapbin

import (
	"bytes"
	"encoding/binary"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func writeNetString(buf *bytes.Buffer, s string) {
	length := len(s)
	for {
		b := byte(length & 0x7f)
		length >>= 7
		if length > 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if length == 0 {
			break
		}
	}
	buf.WriteString(s)
}

func writeInt16(buf *bytes.Buffer, v int16) {
	_ = binary.Write(buf, binary.LittleEndian, v)
}

func writeInt32(buf *bytes.Buffer, v int32) {
	_ = binary.Write(buf, binary.LittleEndian, v)
}

// buildFixtureMeta builds a Gameplay.meta with one data file ("Gameplay0")
// containing two sprites: "tilesets/dirt" at (0,0,2,1) and "decals/test" at (2,0,1,1).
func buildFixtureMeta() []byte {
	buf := &bytes.Buffer{}
	writeInt32(buf, 1)             // version
	writeNetString(buf, "deadbeef") // checksum
	writeInt32(buf, 0)             // unused header field
	writeInt16(buf, 1)             // dataFileCount

	writeNetString(buf, "Gameplay0")
	writeInt16(buf, 2) // spriteCount

	writeNetString(buf, "tilesets/dirt")
	for _, v := range []int16{0, 0, 2, 1, 0, 0, 2, 1} {
		writeInt16(buf, v)
	}

	writeNetString(buf, "decals/test")
	for _, v := range []int16{2, 0, 1, 1, 0, 0, 1, 1} {
		writeInt16(buf, v)
	}

	return buf.Bytes()
}

// buildFixtureData builds a 3x1 .data image (matches the 3-wide sprite sheet
// referenced by buildFixtureMeta): opaque red, opaque green, opaque blue,
// each as its own single-pixel RLE run, with alpha channel present.
func buildFixtureData() []byte {
	buf := &bytes.Buffer{}
	writeInt32(buf, 3) // width
	writeInt32(buf, 1) // height
	buf.WriteByte(1)   // hasAlpha = true

	// pixel 0: red, opaque -> alpha, b, g, r
	buf.WriteByte(1)   // repeat count
	buf.WriteByte(255) // alpha
	buf.WriteByte(0)   // b
	buf.WriteByte(0)   // g
	buf.WriteByte(255) // r

	// pixel 1: green, opaque
	buf.WriteByte(1)
	buf.WriteByte(255)
	buf.WriteByte(0)
	buf.WriteByte(255)
	buf.WriteByte(0)

	// pixel 2: blue, opaque
	buf.WriteByte(1)
	buf.WriteByte(255)
	buf.WriteByte(255)
	buf.WriteByte(0)
	buf.WriteByte(0)

	return buf.Bytes()
}

func TestLoadAtlas_SpriteLookupAndPixelDecode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Gameplay.meta"), buildFixtureMeta(), 0644); err != nil {
		t.Fatalf("failed to write fixture meta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Gameplay0.data"), buildFixtureData(), 0644); err != nil {
		t.Fatalf("failed to write fixture data: %v", err)
	}

	atlas, err := LoadAtlas(dir, "Gameplay.meta")
	if err != nil {
		t.Fatalf("LoadAtlas failed: %v", err)
	}

	if _, _, ok := atlas.GetSprite("unknown/path"); ok {
		t.Errorf("expected unknown sprite path to miss")
	}

	img, meta, ok := atlas.GetSprite("decals/test")
	if !ok {
		t.Fatalf("expected decals/test sprite to be found")
	}
	if meta.Width != 1 || meta.Height != 1 {
		t.Errorf("expected decals/test quad 1x1, got %dx%d", meta.Width, meta.Height)
	}
	got := img.At(2, 0)
	want := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	if got != want {
		t.Errorf("expected decals/test pixel %v, got %v", want, got)
	}

	dirtImg, dirtMeta, ok := atlas.GetSprite("tilesets/dirt")
	if !ok {
		t.Fatalf("expected tilesets/dirt sprite to be found")
	}
	if dirtMeta.Width != 2 || dirtMeta.Height != 1 {
		t.Errorf("expected tilesets/dirt quad 2x1, got %dx%d", dirtMeta.Width, dirtMeta.Height)
	}
	if got := dirtImg.At(0, 0); got != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Errorf("expected tilesets/dirt pixel(0,0) red, got %v", got)
	}
	if got := dirtImg.At(1, 0); got != (color.RGBA{R: 0, G: 255, B: 0, A: 255}) {
		t.Errorf("expected tilesets/dirt pixel(1,0) green, got %v", got)
	}
}

func TestLoadDataImage_TransparentRunAndRepeat(t *testing.T) {
	buf := &bytes.Buffer{}
	writeInt32(buf, 2) // width
	writeInt32(buf, 1) // height
	buf.WriteByte(1)   // hasAlpha

	// pixel 0: fully transparent (alpha=0, rgb bytes omitted per format)
	buf.WriteByte(1)
	buf.WriteByte(0)

	// pixel 1: repeated run of 1 opaque white pixel
	buf.WriteByte(1)
	buf.WriteByte(255)
	buf.WriteByte(255)
	buf.WriteByte(255)
	buf.WriteByte(255)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.data")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write fixture data: %v", err)
	}

	img, err := loadDataImage(path)
	if err != nil {
		t.Fatalf("loadDataImage failed: %v", err)
	}

	if got := img.At(0, 0); got != (color.RGBA{}) {
		t.Errorf("expected transparent pixel, got %v", got)
	}
	if got := img.At(1, 0); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Errorf("expected opaque white pixel, got %v", got)
	}
}
