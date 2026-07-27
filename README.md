# CelesteMapsBinParser

`CelesteMapsBinParser` (`github.com/TheCelesteHub/MapBinsParser`) is a specialized Go CLI tool for parsing and rendering Celeste map `.bin` files (BinaryPacker format).

## Architecture

Built using Cobra CLI framework and structured following SOLID design principles:

```
dependencies/CelesteMapsBinParser/
├── cmd/                        # Cobra CLI subcommand handlers
│   ├── root.go                 # Executable entrypoint & root command
│   ├── count_collectibles.go   # 'count-collectibles' subcommand
│   └── export_map_images.go    # 'export-map-images' subcommand
├── pkg/
│   └── mapbin/                 # Core domain logic
│       ├── bin_reader.go       # BinaryPacker binary parser & varint reader
│       ├── classifier.go       # Entity classifier & regex patterns
│       ├── collectibles.go     # Mod zip/folder map scanner & collectible counter
│       ├── exporter.go         # Map image exporter & manifest generator
│       ├── map_renderer.go     # Room PNG rendering & composite map layout
│       └── types.go            # Data structures & JSON response models
├── testing/                    # Test fixtures & test suite
│   └── mapbin_test.go
├── main.go                     # Go main package
└── go.mod
```

---

## Subcommands & Usage

### 1. `count-collectibles`
Scans map `.bin` files in a mod archive or directory and counts all placed collectibles by type (red, golden, winged golden, moonberry, crystal hearts, mini hearts, silver, speed, rainbow, platinum).

```bash
CelesteMapsBinParser count-collectibles --mod "/path/to/mod.zip"
```

**Flags:**
- `--mod`, `-m` (Required): Path to mod `.zip` file, unpacked folder, or single `.bin` file.

**JSON Output Example:**
```json
{
  "success": true,
  "maps": {
    "1-ForsakenCity": {
      "red": 20,
      "golden": 1,
      "wingedGolden": 1,
      "moon": 0,
      "hearts": 1,
      "miniHearts": 0,
      "silver": 0,
      "speed": 0,
      "rainbow": 0,
      "platinum": 0
    }
  }
}
```

---

### 2. `export-map-images`
Parses room geometries, solid tile grids (8x8), and entity spawn/collectible/hazard locations to generate per-room PNGs and a full-map composite PNG image.

```bash
CelesteMapsBinParser export-map-images --mod "/path/to/mod.zip" --map "1-ForsakenCity" --out "./output_dir"
```

**Flags:**
- `--mod`, `-m` (Required): Path to mod `.zip`, folder, or `.bin` map file.
- `--map`, `-p` (Required): Map SID (e.g. `1-ForsakenCity` or `author/campaign/map`).
- `--out`, `-o` (Required): Directory path to save generated PNG images and `manifest.json`.

**JSON Output Example:**
```json
{
  "success": true,
  "mapSid": "1-ForsakenCity",
  "fullMapPng": "full_map.png",
  "rooms": [
    {
      "name": "lvl_1",
      "x": 0,
      "y": 0,
      "width": 320,
      "height": 180,
      "image": "rooms/room_lvl_1.png"
    }
  ],
  "outDir": "./output_dir"
}
```

---

## Running Tests

Run native Go tests with:
```bash
go test ./...
```
