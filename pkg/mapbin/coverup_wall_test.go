package mapbin

import (
	"bytes"
	"testing"
)

// TestReadFullMapData_CoverupWallRasterizesIntoSolids builds a hand-made .bin
// with one "level" containing an "entities" child with a single coverupWall
// entity at (8,0) sized 16x8 with tiletype="3". It must merge into the room's
// Solids/SolidsTileID grid and never appear in room.Entities (loenn/src/entities/coverup_wall.lua
// confirms coverupWall renders as a blended tile patch, not a generic actor).
func TestReadFullMapData_CoverupWallRasterizesIntoSolids(t *testing.T) {
	data := buildFixtureCoverupWallMap()

	mapData, err := readFullMapData(data)
	if err != nil {
		t.Fatalf("readFullMapData failed: %v", err)
	}
	if len(mapData.Rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(mapData.Rooms))
	}
	room := mapData.Rooms[0]

	if len(room.Entities) != 0 {
		t.Fatalf("expected coverupWall to be rasterized, not appended as an entity, got %d entities", len(room.Entities))
	}

	// entity spans x=[8,24) y=[0,8) -> tile cols 1,2 row 0.
	if !room.Solids[0][1] || !room.Solids[0][2] {
		t.Errorf("expected coverupWall bounds to be marked solid in the tile grid")
	}
	if room.SolidsTileID[0][1] != '3' || room.SolidsTileID[0][2] != '3' {
		t.Errorf("expected coverupWall tiletype '3' to be written into SolidsTileID, got %q/%q", room.SolidsTileID[0][1], room.SolidsTileID[0][2])
	}
	if room.Solids[0][0] {
		t.Errorf("expected cell outside coverupWall bounds to remain empty")
	}
}

func buildFixtureCoverupWallMap() []byte {
	lookup := []string{"map", "level", "name", "x", "y", "width", "height", "entities", "coverupWall", "tiletype"}
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

	coverupWall := &bytes.Buffer{}
	writeInt16(coverupWall, int16(idx["coverupWall"]))
	coverupWall.WriteByte(5) // attrCount: x, y, width, height, tiletype
	writeIntAttr(coverupWall, "x", 8)
	writeIntAttr(coverupWall, "y", 0)
	writeIntAttr(coverupWall, "width", 16)
	writeIntAttr(coverupWall, "height", 8)
	writeStringAttr(coverupWall, "tiletype", "3")
	writeInt16(coverupWall, 0) // childCount

	entities := &bytes.Buffer{}
	writeInt16(entities, int16(idx["entities"]))
	entities.WriteByte(0)   // attrCount
	writeInt16(entities, 1) // childCount
	entities.Write(coverupWall.Bytes())

	level := &bytes.Buffer{}
	writeInt16(level, int16(idx["level"]))
	level.WriteByte(5) // attrCount
	writeStringAttr(level, "name", "lvl_1")
	writeIntAttr(level, "x", 0)
	writeIntAttr(level, "y", 0)
	writeIntAttr(level, "width", 320)
	writeIntAttr(level, "height", 180)
	writeInt16(level, 1) // childCount
	level.Write(entities.Bytes())

	root := &bytes.Buffer{}
	writeNetString(root, "CELESTE MAP")
	writeNetString(root, "package")
	writeInt16(root, int16(len(lookup)))
	for _, s := range lookup {
		writeNetString(root, s)
	}
	writeInt16(root, int16(idx["map"])) // root nameIdx
	root.WriteByte(0)                   // root attrCount
	writeInt16(root, 1)                 // root childCount: level
	root.Write(level.Bytes())

	return root.Bytes()
}
