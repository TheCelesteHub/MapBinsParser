package mapbin

import (
	"bytes"
	"testing"
)

func TestMatchesRoomFilter(t *testing.T) {
	cases := []struct {
		name          string
		roomName      string
		only, exclude string
		wantMatch     bool
	}{
		{"default only-all", "lvl_1", "*", "", true},
		{"exact list match", "b", "a,b,c", "", true},
		{"exact list miss", "d", "a,b,c", "", false},
		{"exclude subtracts", "b", "a,b,c", "b", false},
		{"wildcard prefix", "lvl_intro", "lvl_*", "", true},
		{"wildcard prefix miss", "other", "lvl_*", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := matchesRoomFilter(c.roomName, c.only, c.exclude)
			if got != c.wantMatch {
				t.Errorf("matchesRoomFilter(%q, %q, %q) = %v, want %v", c.roomName, c.only, c.exclude, got, c.wantMatch)
			}
		})
	}
}

// TestReadFullMapData_ParsesParallaxOnlySkipsEffects exercises readFullMapData's
// Style-tree parsing against a hand-built minimal .bin: a root element with an
// empty "level" child and a "Style/Backgrounds" child containing one
// "parallax" backdrop plus one non-parallax "stars" effect element, which
// must be dropped rather than parsed as a backdrop.
func TestReadFullMapData_ParsesParallaxOnlySkipsEffects(t *testing.T) {
	data := buildFixtureStyleMap()

	mapData, err := readFullMapData(data)
	if err != nil {
		t.Fatalf("readFullMapData failed: %v", err)
	}

	if len(mapData.Backdrops) != 1 {
		t.Fatalf("expected exactly 1 backdrop (parallax only), got %d", len(mapData.Backdrops))
	}
	bd := mapData.Backdrops[0]
	if bd.Texture != "bgs/author/bg" {
		t.Errorf("expected texture bgs/author/bg, got %q", bd.Texture)
	}
	if bd.Fg {
		t.Errorf("expected background parallax, got Fg=true")
	}
	if !bd.LoopX || !bd.LoopY {
		t.Errorf("expected loopx/loopy defaults to true, got %v/%v", bd.LoopX, bd.LoopY)
	}
}

func buildFixtureStyleMap() []byte {
	lookup := []string{"map", "level", "name", "x", "y", "width", "height", "Style", "Backgrounds", "parallax", "texture", "stars"}
	idx := make(map[string]int, len(lookup))
	for i, s := range lookup {
		idx[s] = i
	}

	writeStringAttr := func(buf *bytes.Buffer, key, val string) {
		writeInt16(buf, int16(idx[key]))
		buf.WriteByte(6) // type 6: varint-length string
		writeNetString(buf, val)
	}
	writeIntAttr := func(buf *bytes.Buffer, key string, v int32) {
		writeInt16(buf, int16(idx[key]))
		buf.WriteByte(3) // type 3: int32
		writeInt32(buf, v)
	}

	level := &bytes.Buffer{}
	writeInt16(level, int16(idx["level"]))
	level.WriteByte(5) // attrCount
	writeStringAttr(level, "name", "lvl_1")
	writeIntAttr(level, "x", 0)
	writeIntAttr(level, "y", 0)
	writeIntAttr(level, "width", 320)
	writeIntAttr(level, "height", 180)
	writeInt16(level, 0) // childCount

	parallax := &bytes.Buffer{}
	writeInt16(parallax, int16(idx["parallax"]))
	parallax.WriteByte(1) // attrCount
	writeStringAttr(parallax, "texture", "bgs/author/bg")
	writeInt16(parallax, 0) // childCount

	stars := &bytes.Buffer{}
	writeInt16(stars, int16(idx["stars"]))
	stars.WriteByte(0)   // attrCount
	writeInt16(stars, 0) // childCount

	backgrounds := &bytes.Buffer{}
	writeInt16(backgrounds, int16(idx["Backgrounds"]))
	backgrounds.WriteByte(0)   // attrCount
	writeInt16(backgrounds, 2) // childCount
	backgrounds.Write(parallax.Bytes())
	backgrounds.Write(stars.Bytes())

	style := &bytes.Buffer{}
	writeInt16(style, int16(idx["Style"]))
	style.WriteByte(0)   // attrCount
	writeInt16(style, 1) // childCount
	style.Write(backgrounds.Bytes())

	root := &bytes.Buffer{}
	writeNetString(root, "CELESTE MAP")
	writeNetString(root, "package")
	writeInt16(root, int16(len(lookup)))
	for _, s := range lookup {
		writeNetString(root, s)
	}
	writeInt16(root, int16(idx["map"])) // root nameIdx
	root.WriteByte(0)                   // root attrCount
	writeInt16(root, 2)                 // root childCount: level + Style
	root.Write(level.Bytes())
	root.Write(style.Bytes())

	return root.Bytes()
}
