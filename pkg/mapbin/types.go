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
	Name   string  `json:"name"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Kind   string  `json:"kind"`
}

type RoomData struct {
	Name     string        `json:"name"`
	X        int           `json:"x"`
	Y        int           `json:"y"`
	Width    int           `json:"width"`
	Height   int           `json:"height"`
	Solids   [][]bool      `json:"-"`
	Bg       [][]bool      `json:"-"`
	Entities []*EntityData `json:"entities"`
}

type MapRenderData struct {
	Rooms []*RoomData `json:"rooms"`
}

type MapRoomManifestEntry struct {
	Name   string `json:"name"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Image  string `json:"image"`
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
