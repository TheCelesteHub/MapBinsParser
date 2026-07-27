package mapbin

type MapCollectibleCounts struct {
	Red          int `json:"red"`
	Golden       int `json:"golden"`
	WingedGolden int `json:"wingedGolden"`
	Moon         int `json:"moon"`
	Hearts       int `json:"hearts"`
	MiniHearts   int `json:"miniHearts"`
	Silver       int `json:"silver"`
	Speed        int `json:"speed"`
	Rainbow      int `json:"rainbow"`
	Platinum     int `json:"platinum"`
}

type MapCollectiblesResult struct {
	Success bool                             `json:"success"`
	Maps    map[string]*MapCollectibleCounts `json:"maps,omitempty"`
	Failed  map[string]string                `json:"failed,omitempty"`
	Error   string                           `json:"error,omitempty"`
}

type EntityData struct {
	Name     string  `json:"name"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Kind     string  `json:"kind"`
	Moon     bool    `json:"-"`
	Winged   bool    `json:"-"`
	HasNodes bool    `json:"-"`
	TypeAttr string  `json:"-"`
}

type DecalData struct {
	Texture  string  `json:"texture"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	ScaleX   float64 `json:"scaleX"`
	ScaleY   float64 `json:"scaleY"`
	Rotation float64 `json:"rotation"`
	Fg       bool    `json:"fg"`
}

type RoomData struct {
	Name         string        `json:"name"`
	X            int           `json:"x"`
	Y            int           `json:"y"`
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	Solids       [][]bool      `json:"-"`
	Bg           [][]bool      `json:"-"`
	SolidsTileID [][]byte      `json:"-"`
	BgTileID     [][]byte      `json:"-"`
	Entities     []*EntityData `json:"entities"`
	Decals       []*DecalData  `json:"-"`
}

// BackdropData is one <parallax> styleground backdrop, map-wide (Style sits
// outside the room tree in the .bin, as a sibling of the rooms, not nested
// per-level). ScrollX/ScrollY and Blendmode are intentionally not captured:
// scroll only matters with a moving camera (irrelevant to a static per-room
// screenshot), and additive blending is rare enough to just alpha-blend it.
type BackdropData struct {
	Texture string  `json:"texture"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Alpha   float64 `json:"alpha"`
	LoopX   bool    `json:"loopX"`
	LoopY   bool    `json:"loopY"`
	FlipX   bool    `json:"flipX"`
	FlipY   bool    `json:"flipY"`
	Only    string  `json:"only"`
	Exclude string  `json:"exclude"`
	Fg      bool    `json:"fg"`
}

type MapRenderData struct {
	Rooms     []*RoomData     `json:"rooms"`
	Backdrops []*BackdropData `json:"-"`
}

type MapRoomManifestEntry struct {
	Name   string `json:"name"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Image  string `json:"image"`
}

type ExportMapImagesOptions struct {
	GridOnly   bool
	CelesteDir string
}

type ExportMapImagesResult struct {
	Success    bool                    `json:"success"`
	MapSid     string                  `json:"mapSid,omitempty"`
	FullMapPng string                  `json:"fullMapPng,omitempty"`
	Rooms      []*MapRoomManifestEntry `json:"rooms,omitempty"`
	OutDir     string                  `json:"outDir,omitempty"`
	Error      string                  `json:"error,omitempty"`
}

type GenericResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
