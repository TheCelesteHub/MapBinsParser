package mapbin

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var invalidFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9_\-]`)

func sanitizeFilename(s string) string {
	res := invalidFilenameChars.ReplaceAllString(s, "_")
	if res == "" {
		return "unnamed"
	}
	return res
}

func getMapBinBytes(modPath, mapSid string) ([]byte, string, error) {
	info, err := os.Stat(modPath)
	if err != nil {
		return nil, "", err
	}

	if !info.IsDir() && strings.EqualFold(filepath.Ext(modPath), ".bin") {
		data, err := os.ReadFile(modPath)
		if err != nil {
			return nil, "", err
		}
		sid := SidFromMapPath(modPath)
		return data, sid, nil
	}

	if !info.IsDir() {
		reader, err := zip.OpenReader(modPath)
		if err != nil {
			return nil, "", err
		}
		defer reader.Close()

		var matchedFile *zip.File
		for _, file := range reader.File {
			if !IsMapBinPath(file.Name) {
				continue
			}
			sid := SidFromMapPath(file.Name)
			if strings.EqualFold(sid, mapSid) || strings.EqualFold(filepath.Base(sid), mapSid) || mapSid == "" {
				matchedFile = file
				break
			}
		}

		if matchedFile == nil {
			return nil, "", fmt.Errorf("map SID %q not found in zip %s", mapSid, modPath)
		}

		rc, err := matchedFile.Open()
		if err != nil {
			return nil, "", err
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, "", err
		}
		return data, SidFromMapPath(matchedFile.Name), nil
	}

	mapsDir := filepath.Join(modPath, "Maps")
	searchRoot := modPath
	if _, err := os.Stat(mapsDir); err == nil {
		searchRoot = mapsDir
	}

	var foundData []byte
	var foundSid string
	walkErr := filepath.WalkDir(searchRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".bin") {
			return nil
		}
		relative, relErr := filepath.Rel(modPath, path)
		if relErr != nil {
			return nil
		}
		sid := SidFromMapPath(relative)
		if strings.EqualFold(sid, mapSid) || strings.EqualFold(filepath.Base(sid), mapSid) || mapSid == "" {
			d, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			foundData = d
			foundSid = sid
			return io.EOF
		}
		return nil
	})

	if walkErr != nil && walkErr != io.EOF {
		return nil, "", walkErr
	}

	if len(foundData) == 0 {
		return nil, "", fmt.Errorf("map SID %q not found in folder %s", mapSid, modPath)
	}

	return foundData, foundSid, nil
}

func readFullMapData(data []byte) (*MapRenderData, error) {
	r := NewBinReader(data, nil)
	if err := r.ReadHeader(); err != nil {
		return nil, err
	}
	result := &MapRenderData{Rooms: []*RoomData{}}
	if err := r.readFullElement(result, nil, ""); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *BinReader) readFullElement(result *MapRenderData, currentRoom *RoomData, container string) error {
	nameIdx, err := r.u16()
	if err != nil {
		return err
	}
	name, err := r.lookupAt(nameIdx)
	if err != nil {
		return err
	}
	attrCount, err := r.u8()
	if err != nil {
		return err
	}
	attrs, err := r.ReadAllAttributes(int(attrCount))
	if err != nil {
		return err
	}
	childCount, err := r.u16()
	if err != nil {
		return err
	}

	activeRoom := currentRoom

	if name == "level" {
		roomName := getStringAttr(attrs, "name")
		rx := getIntAttr(attrs, "x")
		ry := getIntAttr(attrs, "y")
		rw := getIntAttr(attrs, "width")
		rh := getIntAttr(attrs, "height")
		if rw <= 0 {
			rw = 320
		}
		if rh <= 0 {
			rh = 180
		}
		cols := rw / 8
		rows := rh / 8
		if cols <= 0 {
			cols = 1
		}
		if rows <= 0 {
			rows = 1
		}
		solidsGrid := make([][]bool, rows)
		solidsTileID := make([][]byte, rows)
		for i := range solidsGrid {
			solidsGrid[i] = make([]bool, cols)
			solidsTileID[i] = make([]byte, cols)
		}
		bgGrid := make([][]bool, rows)
		bgTileID := make([][]byte, rows)
		for i := range bgGrid {
			bgGrid[i] = make([]bool, cols)
			bgTileID[i] = make([]byte, cols)
		}

		activeRoom = &RoomData{
			Name:         roomName,
			X:            rx,
			Y:            ry,
			Width:        rw,
			Height:       rh,
			Solids:       solidsGrid,
			Bg:           bgGrid,
			SolidsTileID: solidsTileID,
			BgTileID:     bgTileID,
			Entities:     []*EntityData{},
		}
		result.Rooms = append(result.Rooms, activeRoom)
	} else if activeRoom != nil {
		switch name {
		case "solids":
			tileStr := extractTileString(attrs)
			if tileStr != "" {
				parseTileGrid(tileStr, activeRoom.Solids, activeRoom.SolidsTileID)
			}
		case "bg":
			tileStr := extractTileString(attrs)
			if tileStr != "" {
				parseTileGrid(tileStr, activeRoom.Bg, activeRoom.BgTileID)
			}
		case "Style":
		default:
			if container == "entities" {
				ex := getFloatAttr(attrs, "x")
				ey := getFloatAttr(attrs, "y")
				ew := getFloatAttr(attrs, "width")
				eh := getFloatAttr(attrs, "height")
				moon := getBoolAttr(attrs, "moon")
				winged := getBoolAttr(attrs, "winged")

				kind := "generic"
				if name == "player" || (SpawnPattern.MatchString(name) && !SuffixDenyPattern.MatchString(name)) {
					kind = "spawn"
				} else if IsCollectibleEntity(name, moon, winged) {
					kind = "collectible"
				} else if HazardPattern.MatchString(name) {
					kind = "hazard"
				}

				activeRoom.Entities = append(activeRoom.Entities, &EntityData{
					Name:   name,
					X:      ex,
					Y:      ey,
					Width:  ew,
					Height: eh,
					Kind:   kind,
				})
			} else if container == "fgdecals" || container == "bgdecals" {
				activeRoom.Decals = append(activeRoom.Decals, &DecalData{
					Texture:  getStringAttr(attrs, "texture"),
					X:        getFloatAttr(attrs, "x"),
					Y:        getFloatAttr(attrs, "y"),
					ScaleX:   getFloatAttrDefault(attrs, "scaleX", 1),
					ScaleY:   getFloatAttrDefault(attrs, "scaleY", 1),
					Rotation: getFloatAttr(attrs, "rotation"),
					Fg:       container == "fgdecals",
				})
			}
		}
	}

	childContainer := container
	if name == "entities" || name == "triggers" || name == "fgdecals" || name == "bgdecals" {
		childContainer = name
	}

	for i := 0; i < childCount; i++ {
		if err := r.readFullElement(result, activeRoom, childContainer); err != nil {
			return err
		}
	}

	return nil
}

func getIntAttr(attrs map[string]interface{}, key string) int {
	if val, ok := attrs[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case bool:
			if v {
				return 1
			}
			return 0
		}
	}
	return 0
}

func getFloatAttr(attrs map[string]interface{}, key string) float64 {
	if val, ok := attrs[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		}
	}
	return 0.0
}

func getFloatAttrDefault(attrs map[string]interface{}, key string, def float64) float64 {
	if val, ok := attrs[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		}
	}
	return def
}

func getStringAttr(attrs map[string]interface{}, key string) string {
	if val, ok := attrs[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func getBoolAttr(attrs map[string]interface{}, key string) bool {
	if val, ok := attrs[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func extractTileString(attrs map[string]interface{}) string {
	if val, ok := attrs["innerText"]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	for _, val := range attrs {
		if s, ok := val.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func parseTileGrid(tileStr string, grid [][]bool, tileIDGrid [][]byte) {
	rows := len(grid)
	if rows == 0 {
		return
	}
	cols := len(grid[0])
	lines := strings.Split(strings.ReplaceAll(tileStr, "\r\n", "\n"), "\n")
	for r := 0; r < rows; r++ {
		if r >= len(lines) {
			break
		}
		line := lines[r]
		for c := 0; c < cols; c++ {
			if c >= len(line) {
				break
			}
			ch := line[c]
			grid[r][c] = (ch != '0' && ch != ' ')
			if tileIDGrid != nil && ch != ' ' {
				tileIDGrid[r][c] = ch
			}
		}
	}
}

func ExportMapImages(modPath, mapSid, outDir string, opts ExportMapImagesOptions) (*ExportMapImagesResult, error) {
	data, actualSid, err := getMapBinBytes(modPath, mapSid)
	if err != nil {
		return nil, err
	}

	mapData, err := readFullMapData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse map data: %w", err)
	}

	var assets *RenderAssets
	if !opts.GridOnly && opts.CelesteDir != "" {
		loaded, loadErr := LoadRenderAssets(opts.CelesteDir)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "warning: real-asset rendering unavailable (%v), falling back to grid rendering\n", loadErr)
		} else {
			assets = loaded
		}
	}

	roomsDir := filepath.Join(outDir, "rooms")
	if err := os.MkdirAll(roomsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rooms output dir: %w", err)
	}

	usedNames := make(map[string]int)
	manifestRooms := make([]*MapRoomManifestEntry, 0, len(mapData.Rooms))

	for _, room := range mapData.Rooms {
		baseName := sanitizeFilename(room.Name)
		count := usedNames[baseName]
		usedNames[baseName] = count + 1
		fileName := baseName
		if count > 0 {
			fileName = fmt.Sprintf("%s_%d", baseName, count+1)
		}

		relPath := filepath.ToSlash(filepath.Join("rooms", fmt.Sprintf("room_%s.png", fileName)))
		roomPngPath := filepath.Join(outDir, relPath)

		roomImg := RenderRoomToImage(room, assets)
		f, createErr := os.Create(roomPngPath)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create room PNG %s: %w", roomPngPath, createErr)
		}
		if encodeErr := png.Encode(f, roomImg); encodeErr != nil {
			f.Close()
			return nil, fmt.Errorf("failed to encode room PNG %s: %w", roomPngPath, encodeErr)
		}
		f.Close()

		manifestRooms = append(manifestRooms, &MapRoomManifestEntry{
			Name:   room.Name,
			X:      room.X,
			Y:      room.Y,
			Width:  room.Width,
			Height: room.Height,
			Image:  relPath,
		})
	}

	fullImg := RenderFullMapComposite(mapData.Rooms, assets)
	fullPngRel := "full_map.png"
	fullPngPath := filepath.Join(outDir, fullPngRel)
	f, createErr := os.Create(fullPngPath)
	if createErr != nil {
		return nil, fmt.Errorf("failed to create full map PNG %s: %w", fullPngPath, createErr)
	}
	if encodeErr := png.Encode(f, fullImg); encodeErr != nil {
		f.Close()
		return nil, fmt.Errorf("failed to encode full map PNG %s: %w", fullPngPath, encodeErr)
	}
	f.Close()

	result := &ExportMapImagesResult{
		Success:    true,
		MapSid:     actualSid,
		FullMapPng: fullPngRel,
		Rooms:      manifestRooms,
		OutDir:     outDir,
	}

	manifestBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest JSON: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "manifest.json"), manifestBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write manifest.json: %w", err)
	}

	return result, nil
}
