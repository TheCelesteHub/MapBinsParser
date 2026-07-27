package mapbin

import (
	"os"
	"path/filepath"
	"testing"
)

const fixtureTilesetXML = `<Data>
  <Tileset id="1" path="dirt" ignores="">
    <set mask="000-010-000" tiles="9,9"/>
    <set mask="111-111-111" tiles="0,0"/>
    <set mask="center" tiles="1,1"/>
  </Tileset>
</Data>`

func writeFixtureTilesetXML(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "ForegroundTiles.xml")
	if err := os.WriteFile(path, []byte(fixtureTilesetXML), 0644); err != nil {
		t.Fatalf("failed to write fixture xml: %v", err)
	}
	return path
}

func TestLoadTilesetXML_ParsesTilesetsAndSets(t *testing.T) {
	path := writeFixtureTilesetXML(t)
	rules, err := LoadTilesetXML(path)
	if err != nil {
		t.Fatalf("LoadTilesetXML failed: %v", err)
	}

	rule, ok := rules.byID['1']
	if !ok {
		t.Fatalf("expected tileset id '1' to be parsed")
	}
	if rule.Path != "dirt" {
		t.Errorf("expected path 'dirt', got %q", rule.Path)
	}
	if len(rule.Rules) != 2 {
		t.Fatalf("expected 2 non-special mask rules, got %d", len(rule.Rules))
	}
}

func TestGetTileQuad_MatchesFirstRuleAndFallsBackToCenter(t *testing.T) {
	path := writeFixtureTilesetXML(t)
	rules, err := LoadTilesetXML(path)
	if err != nil {
		t.Fatalf("LoadTilesetXML failed: %v", err)
	}

	// isolated tile: air on all sides except none - matches mask "111-111-111"? no,
	// with no neighbors filled it should match "000-000-000" style all-air rule instead.
	grid := [][]byte{
		{0, 0, 0},
		{0, '1', 0},
		{0, 0, 0},
	}
	tsPath, quad, ok := GetTileQuad(rules, grid, 1, 1)
	if !ok {
		t.Fatalf("expected a quad match for isolated tile")
	}
	if tsPath != "dirt" {
		t.Errorf("expected tileset path 'dirt', got %q", tsPath)
	}
	if quad != (TileQuad{Col: 9, Row: 9}) {
		t.Errorf("expected isolated tile to match the all-air mask rule quad {9,9}, got %v", quad)
	}

	// fully surrounded tile: all 8 neighbors filled with tile '1'.
	fullGrid := [][]byte{
		{'1', '1', '1'},
		{'1', '1', '1'},
		{'1', '1', '1'},
	}
	tsPath2, quad2, ok := GetTileQuad(rules, fullGrid, 1, 1)
	if !ok {
		t.Fatalf("expected a quad match for surrounded tile")
	}
	if tsPath2 != "dirt" {
		t.Errorf("expected tileset path 'dirt', got %q", tsPath2)
	}
	if quad2 != (TileQuad{Col: 0, Row: 0}) {
		t.Errorf("expected surrounded tile to match the all-filled mask rule quad {0,0}, got %v", quad2)
	}
}

// TestGetTileQuad_UncoveredNeighborPatternMisses reproduces a 2x2 solid block:
// the corner tile has two filled cardinal neighbors (right, down) and two air
// (up, left) - not covered by any explicit rule, not fully-surrounded, no
// padding defined. Must miss (ok=false), not silently fall back to an
// arbitrary rule's quad (the bug that made thick solids render as a wrong
// flat-looking blob instead of the honest flat-color fallback).
func TestGetTileQuad_UncoveredNeighborPatternMisses(t *testing.T) {
	path := writeFixtureTilesetXML(t)
	rules, err := LoadTilesetXML(path)
	if err != nil {
		t.Fatalf("LoadTilesetXML failed: %v", err)
	}

	grid := [][]byte{
		{'1', '1'},
		{'1', '1'},
	}
	if _, _, ok := GetTileQuad(rules, grid, 0, 0); ok {
		t.Errorf("expected uncovered 2x2-corner neighbor pattern to miss, not fall back to an arbitrary rule quad")
	}
}

func TestGetTileQuad_UnknownTileIDMisses(t *testing.T) {
	path := writeFixtureTilesetXML(t)
	rules, err := LoadTilesetXML(path)
	if err != nil {
		t.Fatalf("LoadTilesetXML failed: %v", err)
	}

	grid := [][]byte{{'z'}}
	if _, _, ok := GetTileQuad(rules, grid, 0, 0); ok {
		t.Errorf("expected unknown tile id to miss")
	}
}
