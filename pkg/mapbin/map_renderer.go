package mapbin

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"path/filepath"
)

var (
	ColorRoomBg        = color.RGBA{R: 24, G: 24, B: 28, A: 255}
	ColorSolidTile     = color.RGBA{R: 74, G: 85, B: 104, A: 255}
	ColorRoomBorder    = color.RGBA{R: 60, G: 64, B: 72, A: 255}
	ColorSpawnMarker   = color.RGBA{R: 72, G: 187, B: 120, A: 255}
	ColorCollectMarker = color.RGBA{R: 236, G: 201, B: 75, A: 255}
	ColorHazardMarker  = color.RGBA{R: 245, G: 101, B: 101, A: 255}
	ColorGenericMarker = color.RGBA{R: 102, G: 126, B: 234, A: 200}
	ColorFullMapBg     = color.RGBA{R: 10, G: 10, B: 12, A: 255}
)

func FillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	bounds := img.Bounds()
	if x0 < bounds.Min.X {
		x0 = bounds.Min.X
	}
	if y0 < bounds.Min.Y {
		y0 = bounds.Min.Y
	}
	if x1 > bounds.Max.X {
		x1 = bounds.Max.X
	}
	if y1 > bounds.Max.Y {
		y1 = bounds.Max.Y
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.Set(x, y, c)
		}
	}
}

func DrawRectBorder(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	FillRect(img, x0, y0, x1, y0+1, c)
	FillRect(img, x0, y1-1, x1, y1, c)
	FillRect(img, x0, y0, x0+1, y1, c)
	FillRect(img, x1-1, y0, x1, y1, c)
}

// RenderAssets bundles the real Celeste tile/decal assets needed to render
// pixel-accurate rooms (as opposed to the flat-color fallback). Pass nil to
// force flat-color rendering.
type RenderAssets struct {
	FgTileset *TilesetRules
	BgTileset *TilesetRules
	Atlas     *Atlas
}

// LoadRenderAssets loads ForegroundTiles.xml, BackgroundTiles.xml, and the
// Gameplay atlas from a Celeste install root (the folder containing Content/).
func LoadRenderAssets(celesteDir string) (*RenderAssets, error) {
	graphicsDir := filepath.Join(celesteDir, "Content", "Graphics")

	fg, err := LoadTilesetXML(filepath.Join(graphicsDir, "ForegroundTiles.xml"))
	if err != nil {
		return nil, fmt.Errorf("load ForegroundTiles.xml: %w", err)
	}
	bg, err := LoadTilesetXML(filepath.Join(graphicsDir, "BackgroundTiles.xml"))
	if err != nil {
		return nil, fmt.Errorf("load BackgroundTiles.xml: %w", err)
	}
	atlas, err := LoadAtlas(filepath.Join(graphicsDir, "Atlases"), "Gameplay.meta")
	if err != nil {
		return nil, fmt.Errorf("load Gameplay.meta: %w", err)
	}

	return &RenderAssets{FgTileset: fg, BgTileset: bg, Atlas: atlas}, nil
}

func drawRealTile(img *image.RGBA, tileIDGrid [][]byte, tileset *TilesetRules, atlas *Atlas, row, col int) bool {
	tsPath, quad, ok := GetTileQuad(tileset, tileIDGrid, row, col)
	if !ok {
		return false
	}
	sprite, meta, ok := atlas.GetSprite("tilesets/" + tsPath)
	if !ok {
		return false
	}
	srcX := meta.X + quad.Col*8
	srcY := meta.Y + quad.Row*8
	dstX := col * 8
	dstY := row * 8
	draw.Draw(img, image.Rect(dstX, dstY, dstX+8, dstY+8), sprite, image.Point{X: srcX, Y: srcY}, draw.Over)
	return true
}

func drawTileLayer(img *image.RGBA, tileIDGrid [][]byte, tileset *TilesetRules, atlas *Atlas) {
	for r := range tileIDGrid {
		for c := range tileIDGrid[r] {
			drawRealTile(img, tileIDGrid, tileset, atlas, r, c)
		}
	}
}

func RenderRoomToImage(room *RoomData, assets *RenderAssets) *image.RGBA {
	w := room.Width
	h := room.Height
	if w <= 0 {
		w = 320
	}
	if h <= 0 {
		h = 180
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	FillRect(img, 0, 0, w, h, ColorRoomBg)

	if assets != nil && assets.BgTileset != nil && assets.Atlas != nil {
		drawTileLayer(img, room.BgTileID, assets.BgTileset, assets.Atlas)
	}

	rows := len(room.Solids)
	if rows > 0 {
		cols := len(room.Solids[0])
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				if !room.Solids[r][c] {
					continue
				}
				drawn := false
				if assets != nil && assets.FgTileset != nil && assets.Atlas != nil {
					drawn = drawRealTile(img, room.SolidsTileID, assets.FgTileset, assets.Atlas, r, c)
				}
				if !drawn {
					FillRect(img, c*8, r*8, (c+1)*8, (r+1)*8, ColorSolidTile)
				}
			}
		}
	}

	for _, ent := range room.Entities {
		ex := int(ent.X)
		ey := int(ent.Y)
		ew := int(ent.Width)
		eh := int(ent.Height)
		if ew <= 0 {
			ew = 8
		}
		if eh <= 0 {
			eh = 8
		}

		var markerColor color.Color
		switch ent.Kind {
		case "spawn":
			markerColor = ColorSpawnMarker
			if ew < 8 {
				ew = 8
			}
			if eh < 12 {
				eh = 12
			}
		case "collectible":
			markerColor = ColorCollectMarker
		case "hazard":
			markerColor = ColorHazardMarker
		default:
			markerColor = ColorGenericMarker
		}

		FillRect(img, ex, ey, ex+ew, ey+eh, markerColor)
	}

	DrawRectBorder(img, 0, 0, w, h, ColorRoomBorder)
	return img
}

func RenderFullMapComposite(rooms []*RoomData, assets *RenderAssets) *image.RGBA {
	if len(rooms) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}

	minX := rooms[0].X
	minY := rooms[0].Y
	maxX := rooms[0].X + rooms[0].Width
	maxY := rooms[0].Y + rooms[0].Height

	for _, r := range rooms {
		if r.X < minX {
			minX = r.X
		}
		if r.Y < minY {
			minY = r.Y
		}
		if r.X+r.Width > maxX {
			maxX = r.X + r.Width
		}
		if r.Y+r.Height > maxY {
			maxY = r.Y + r.Height
		}
	}

	totalW := maxX - minX
	totalH := maxY - minY
	if totalW <= 0 {
		totalW = 1
	}
	if totalH <= 0 {
		totalH = 1
	}

	fullImg := image.NewRGBA(image.Rect(0, 0, totalW, totalH))
	FillRect(fullImg, 0, 0, totalW, totalH, ColorFullMapBg)

	for _, room := range rooms {
		roomImg := RenderRoomToImage(room, assets)
		offsetX := room.X - minX
		offsetY := room.Y - minY
		dstRect := image.Rect(offsetX, offsetY, offsetX+room.Width, offsetY+room.Height)
		draw.Draw(fullImg, dstRect, roomImg, image.Point{0, 0}, draw.Over)
		DrawRectBorder(fullImg, offsetX, offsetY, offsetX+room.Width, offsetY+room.Height, ColorRoomBorder)
	}

	return fullImg
}
