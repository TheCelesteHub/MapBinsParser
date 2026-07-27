package mapbin

import (
	"encoding/xml"
	"os"
	"strconv"
	"strings"
)

// TileQuad is one "col,row" 8px cell position within a tileset's atlas image.
type TileQuad struct {
	Col, Row int
}

// maskCell is one of the 9 cells of a 3x3 neighbor mask: required-filled,
// required-air, or wildcard (matches either).
type maskCell int

const (
	maskAir maskCell = iota
	maskFilled
	maskWildcard
)

// TileMaskRule is one <set mask="..." tiles="..."/> rule: match this tileset's
// neighbors against Mask (center cell, index 4, is always ignored — a tile
// only ever autotiles against itself), and if it matches, render one of Quads.
type TileMaskRule struct {
	Mask  [9]maskCell
	Quads []TileQuad
}

// TilesetRule holds one <Tileset> entry's parsed autotiling rules.
type TilesetRule struct {
	ID      byte
	Path    string
	Ignores map[byte]bool
	Rules   []TileMaskRule
	Padding []TileQuad
	Center  []TileQuad
}

// TilesetRules is a loaded ForegroundTiles.xml/BackgroundTiles.xml rule set, keyed by tile-id char.
type TilesetRules struct {
	byID map[byte]*TilesetRule
}

type xmlTilesetData struct {
	Tilesets []xmlTileset `xml:"Tileset"`
}

type xmlTileset struct {
	ID      string    `xml:"id,attr"`
	Path    string    `xml:"path,attr"`
	Copy    string    `xml:"copy,attr"`
	Ignores string    `xml:"ignores,attr"`
	Sets    []xmlSet  `xml:"set"`
}

type xmlSet struct {
	Mask  string `xml:"mask,attr"`
	Tiles string `xml:"tiles,attr"`
}

func parseMask(s string) ([9]maskCell, bool) {
	var mask [9]maskCell
	rows := strings.Split(s, "-")
	if len(rows) != 3 {
		return mask, false
	}
	idx := 0
	for _, row := range rows {
		if len(row) != 3 {
			return mask, false
		}
		for _, ch := range row {
			switch ch {
			case '1':
				mask[idx] = maskFilled
			case '0':
				mask[idx] = maskAir
			case 'x', 'X':
				mask[idx] = maskWildcard
			default:
				return mask, false
			}
			idx++
		}
	}
	return mask, true
}

func parseQuads(s string) []TileQuad {
	quads := make([]TileQuad, 0, 1)
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		coords := strings.Split(part, ",")
		if len(coords) != 2 {
			continue
		}
		col, errCol := strconv.Atoi(strings.TrimSpace(coords[0]))
		row, errRow := strconv.Atoi(strings.TrimSpace(coords[1]))
		if errCol != nil || errRow != nil {
			continue
		}
		quads = append(quads, TileQuad{Col: col, Row: row})
	}
	return quads
}

// LoadTilesetXML parses a ForegroundTiles.xml/BackgroundTiles.xml rule file
// (schema shared by Ahorn and Loenn, see docs/TheCelesteDesktop/Loenn.md).
func LoadTilesetXML(path string) (*TilesetRules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var parsed xmlTilesetData
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}

	rules := &TilesetRules{byID: make(map[byte]*TilesetRule)}
	for _, ts := range parsed.Tilesets {
		if len(ts.ID) == 0 {
			continue
		}
		rule := &TilesetRule{
			ID:      ts.ID[0],
			Path:    ts.Path,
			Ignores: make(map[byte]bool),
		}
		for _, ignoreID := range strings.Split(ts.Ignores, ",") {
			ignoreID = strings.TrimSpace(ignoreID)
			if ignoreID != "" {
				rule.Ignores[ignoreID[0]] = true
			}
		}
		for _, set := range ts.Sets {
			quads := parseQuads(set.Tiles)
			switch strings.ToLower(set.Mask) {
			case "padding":
				rule.Padding = quads
			case "center":
				rule.Center = quads
			default:
				if mask, ok := parseMask(set.Mask); ok {
					rule.Rules = append(rule.Rules, TileMaskRule{Mask: mask, Quads: quads})
				}
			}
		}
		rules.byID[rule.ID] = rule
	}

	// ponytail: copy="X" inheritance appends the referenced tileset's rules as a
	// fallback after this tileset's own (own rules still win first-match-wins);
	// doesn't handle chained copies (A copies B copies C) — extend if a real map needs it.
	for _, ts := range parsed.Tilesets {
		if len(ts.ID) == 0 || ts.Copy == "" {
			continue
		}
		rule := rules.byID[ts.ID[0]]
		source, ok := rules.byID[ts.Copy[0]]
		if rule == nil || !ok || source == rule {
			continue
		}
		rule.Rules = append(rule.Rules, source.Rules...)
		if len(rule.Padding) == 0 {
			rule.Padding = source.Padding
		}
		if len(rule.Center) == 0 {
			rule.Center = source.Center
		}
	}

	return rules, nil
}

func isTileFilled(id byte, ignores map[byte]bool) bool {
	if id == 0 || id == '0' || id == ' ' {
		return false
	}
	return !ignores[id]
}

func tileIDAt(grid [][]byte, row, col int) byte {
	if row < 0 || row >= len(grid) || col < 0 || col >= len(grid[row]) {
		return 0
	}
	return grid[row][col]
}

// GetTileQuad resolves the tileset path + tile quad to render for grid[row][col],
// by matching its 3x3 neighborhood (center cell excluded) against the tileset's
// mask rules in document order, falling back to Center/Padding quads.
func GetTileQuad(rules *TilesetRules, grid [][]byte, row, col int) (path string, quad TileQuad, ok bool) {
	id := tileIDAt(grid, row, col)
	if id == 0 || id == '0' || id == ' ' {
		return "", TileQuad{}, false
	}
	rule, exists := rules.byID[id]
	if !exists {
		return "", TileQuad{}, false
	}

	neighbors := [9]bool{}
	i := 0
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			neighbors[i] = isTileFilled(tileIDAt(grid, row+dr, col+dc), rule.Ignores)
			i++
		}
	}

	for _, r := range rule.Rules {
		if maskMatches(r.Mask, neighbors) {
			if q, ok := pickQuad(r.Quads, row, col); ok {
				return rule.Path, q, true
			}
		}
	}

	if isSurroundedByFilled(neighbors) {
		if q, ok := pickQuad(rule.Center, row, col); ok {
			return rule.Path, q, true
		}
	}
	if q, ok := pickQuad(rule.Padding, row, col); ok {
		return rule.Path, q, true
	}
	if len(rule.Rules) > 0 {
		if q, ok := pickQuad(rule.Rules[0].Quads, row, col); ok {
			return rule.Path, q, true
		}
	}
	return "", TileQuad{}, false
}

func maskMatches(mask [9]maskCell, neighbors [9]bool) bool {
	for i := 0; i < 9; i++ {
		if i == 4 {
			continue // center is never checked - a tile only autotiles against itself
		}
		switch mask[i] {
		case maskWildcard:
			continue
		case maskFilled:
			if !neighbors[i] {
				return false
			}
		case maskAir:
			if neighbors[i] {
				return false
			}
		}
	}
	return true
}

// isSurroundedByFilled reports whether all 4 cardinal neighbors are filled -
// a simplified stand-in for Loenn's "checkPadding" (which also looks 2 cells
// out); ponytail: good enough to pick center vs. edge tiles, doesn't reproduce
// wide-padding tilesets exactly.
func isSurroundedByFilled(neighbors [9]bool) bool {
	return neighbors[1] && neighbors[3] && neighbors[5] && neighbors[7]
}

func pickQuad(quads []TileQuad, row, col int) (TileQuad, bool) {
	if len(quads) == 0 {
		return TileQuad{}, false
	}
	idx := (row*31 + col*17) % len(quads)
	if idx < 0 {
		idx += len(quads)
	}
	return quads[idx], true
}
