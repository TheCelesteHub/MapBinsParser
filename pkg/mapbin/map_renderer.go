package mapbin

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"path/filepath"
	"regexp"
	"strings"
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

// drawDecal blits a decal at its natural sprite size, centered on (X, Y).
// ponytail: only mirroring (sign of scaleX/scaleY) and no rotation is
// applied - real Celeste decals can be magnitude-scaled/rotated too, but
// the vast majority aren't; extend here if a real map needs more than a flip.
func drawDecal(img *image.RGBA, decal *DecalData, atlas *Atlas) {
	sprite, meta, ok := atlas.GetSprite("decals/" + decal.Texture)
	if !ok {
		return
	}
	w := meta.RealWidth
	h := meta.RealHeight
	if w <= 0 || h <= 0 {
		return
	}
	dstX := int(decal.X) - w/2
	dstY := int(decal.Y) - h/2
	flipX := decal.ScaleX < 0
	flipY := decal.ScaleY < 0
	bounds := img.Bounds()

	for y := 0; y < h; y++ {
		sy := meta.Y + y
		if flipY {
			sy = meta.Y + (h - 1 - y)
		}
		for x := 0; x < w; x++ {
			sx := meta.X + x
			if flipX {
				sx = meta.X + (w - 1 - x)
			}
			px, py := dstX+x, dstY+y
			if px < bounds.Min.X || px >= bounds.Max.X || py < bounds.Min.Y || py >= bounds.Max.Y {
				continue
			}
			c := sprite.At(sx, sy)
			if _, _, _, a := c.RGBA(); a == 0 {
				continue
			}
			img.Set(px, py, c)
		}
	}
}

func drawDecalLayer(img *image.RGBA, decals []*DecalData, fg bool, atlas *Atlas) {
	for _, d := range decals {
		if d.Fg == fg {
			drawDecal(img, d, atlas)
		}
	}
}

// globToRegexp converts a "*"-wildcard room-name pattern into an anchored,
// case-insensitive regexp (the only wildcard syntax Celeste stylegrounds use).
func globToRegexp(pattern string) *regexp.Regexp {
	escaped := regexp.QuoteMeta(pattern)
	escaped = strings.ReplaceAll(escaped, `\*`, ".*")
	re, err := regexp.Compile("(?i)^" + escaped + "$")
	if err != nil {
		return regexp.MustCompile("^$")
	}
	return re
}

// matchesRoomFilter reports whether a backdrop's only/exclude comma-separated
// glob pattern lists apply to roomName.
func matchesRoomFilter(roomName, only, exclude string) bool {
	matched := false
	for _, pattern := range strings.Split(only, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" && globToRegexp(pattern).MatchString(roomName) {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}

	for _, pattern := range strings.Split(exclude, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" && globToRegexp(pattern).MatchString(roomName) {
			return false
		}
	}
	return true
}

// drawParallaxLayer blits each matching backdrop's texture, tiled across the
// room bounds when LoopX/LoopY, starting at (X, Y). scrollx/scrolly and
// blendmode are deliberately not modeled - see BackdropData's doc comment.
func drawParallaxLayer(img *image.RGBA, roomName string, roomW, roomH int, backdrops []*BackdropData, fg bool, atlas *Atlas) {
	for _, bd := range backdrops {
		if bd.Fg != fg || !matchesRoomFilter(roomName, bd.Only, bd.Exclude) {
			continue
		}
		sprite, meta, ok := atlas.GetSprite(bd.Texture)
		if !ok {
			continue
		}
		w, h := meta.Width, meta.Height
		if w <= 0 || h <= 0 {
			continue
		}

		startX := int(bd.X)
		startY := int(bd.Y)
		if bd.LoopX {
			startX = ((startX % w) + w) % w
			startX -= w
		}
		if bd.LoopY {
			startY = ((startY % h) + h) % h
			startY -= h
		}

		for oy := startY; oy < roomH; oy += h {
			for ox := startX; ox < roomW; ox += w {
				drawParallaxTile(img, sprite, meta, ox, oy, bd)
				if !bd.LoopX {
					break
				}
			}
			if !bd.LoopY {
				break
			}
		}
	}
}

func drawParallaxTile(img *image.RGBA, sprite image.Image, meta SpriteMeta, dstX, dstY int, bd *BackdropData) {
	bounds := img.Bounds()
	for y := 0; y < meta.Height; y++ {
		sy := meta.Y + y
		if bd.FlipY {
			sy = meta.Y + (meta.Height - 1 - y)
		}
		for x := 0; x < meta.Width; x++ {
			sx := meta.X + x
			if bd.FlipX {
				sx = meta.X + (meta.Width - 1 - x)
			}
			px, py := dstX+x, dstY+y
			if px < bounds.Min.X || px >= bounds.Max.X || py < bounds.Min.Y || py >= bounds.Max.Y {
				continue
			}
			r, g, b, a := sprite.At(sx, sy).RGBA()
			if a == 0 {
				continue
			}
			if bd.Alpha < 1 {
				a = uint32(float64(a) * bd.Alpha)
				r = uint32(float64(r) * bd.Alpha)
				g = uint32(float64(g) * bd.Alpha)
				b = uint32(float64(b) * bd.Alpha)
			}
			img.Set(px, py, color.RGBA64{R: uint16(r), G: uint16(g), B: uint16(b), A: uint16(a)})
		}
	}
}

func RenderRoomToImage(room *RoomData, assets *RenderAssets, backdrops []*BackdropData) *image.RGBA {
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

	if assets != nil && assets.Atlas != nil {
		drawParallaxLayer(img, room.Name, w, h, backdrops, false, assets.Atlas)
	}
	if assets != nil && assets.BgTileset != nil && assets.Atlas != nil {
		drawTileLayer(img, room.BgTileID, assets.BgTileset, assets.Atlas)
	}
	if assets != nil && assets.Atlas != nil {
		drawDecalLayer(img, room.Decals, false, assets.Atlas)
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

	if assets != nil && assets.Atlas != nil {
		drawDecalLayer(img, room.Decals, true, assets.Atlas)
		drawParallaxLayer(img, room.Name, w, h, backdrops, true, assets.Atlas)
	}

	DrawRectBorder(img, 0, 0, w, h, ColorRoomBorder)
	return img
}

func RenderFullMapComposite(rooms []*RoomData, assets *RenderAssets, backdrops []*BackdropData) *image.RGBA {
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
		roomImg := RenderRoomToImage(room, assets, backdrops)
		offsetX := room.X - minX
		offsetY := room.Y - minY
		dstRect := image.Rect(offsetX, offsetY, offsetX+room.Width, offsetY+room.Height)
		draw.Draw(fullImg, dstRect, roomImg, image.Point{0, 0}, draw.Over)
		DrawRectBorder(fullImg, offsetX, offsetY, offsetX+room.Width, offsetY+room.Height, ColorRoomBorder)
	}

	return fullImg
}
