package mapbin

import (
	"archive/zip"
	"bytes"
	"image"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// modAssetResolver resolves a mod's own loose Graphics/ files (loose PNGs
// under Graphics/Atlases/Gameplay/, and Graphics/ForegroundTiles.xml /
// BackgroundTiles.xml overrides), whether the mod is an unpacked folder or a
// zip - the same two shapes getMapBinBytes already handles. Everest lets mods
// ship these paths as an overlay on top of the base game's packed atlas; see
// docs/Everest/ModStructure.md ("Adding Graphics" / "File Layout" sections).
type modAssetResolver struct {
	dir      string // set when the mod is an unpacked folder
	zip      *zip.ReadCloser
	zipIndex map[string]*zip.File // lowercased relative path -> entry, set when the mod is a .zip
	pngCache map[string]image.Image
}

// newModAssetResolver builds a resolver for modPath (same value passed as
// --mod). Returns (nil, nil) if modPath isn't a real dir/zip (e.g. a bare
// .bin file mod path) - no error, just "no overlay available", matching the
// rest of this pipeline's silent-fallback convention. The returned closer
// (non-nil only for the zip case) must be deferred by the caller.
func newModAssetResolver(modPath string) (*modAssetResolver, func()) {
	info, err := os.Stat(modPath)
	if err != nil {
		return nil, nil
	}

	if info.IsDir() {
		return &modAssetResolver{dir: modPath, pngCache: make(map[string]image.Image)}, nil
	}

	if !strings.EqualFold(filepath.Ext(modPath), ".zip") {
		return nil, nil
	}

	reader, err := zip.OpenReader(modPath)
	if err != nil {
		return nil, nil
	}

	index := make(map[string]*zip.File, len(reader.File))
	for _, f := range reader.File {
		index[strings.ToLower(filepath.ToSlash(f.Name))] = f
	}

	resolver := &modAssetResolver{zip: reader, zipIndex: index, pngCache: make(map[string]image.Image)}
	return resolver, func() { reader.Close() }
}

// readBytes reads relPath (e.g. "Graphics/ForegroundTiles.xml") from the mod.
func (m *modAssetResolver) readBytes(relPath string) ([]byte, bool) {
	if m.dir != "" {
		data, err := os.ReadFile(filepath.Join(m.dir, relPath))
		if err != nil {
			return nil, false
		}
		return data, true
	}

	entry, ok := m.zipIndex[strings.ToLower(relPath)]
	if !ok {
		return nil, false
	}
	rc, err := entry.Open()
	if err != nil {
		return nil, false
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, false
	}
	return data, true
}

// getSprite decodes Graphics/Atlases/Gameplay/<atlasPath>.png, memoized. Loose
// mod PNGs are standalone images, not sub-rects of a shared sheet - the caller
// (Atlas.GetSprite) synthesizes SpriteMeta from the decoded image's own bounds.
func (m *modAssetResolver) getSprite(atlasPath string) (image.Image, bool) {
	if cached, ok := m.pngCache[atlasPath]; ok {
		return cached, true
	}

	data, ok := m.readBytes("Graphics/Atlases/Gameplay/" + atlasPath + ".png")
	if !ok {
		return nil, false
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}

	m.pngCache[atlasPath] = img
	return img, true
}
