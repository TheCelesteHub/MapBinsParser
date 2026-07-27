package mapbin

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

// TestResolveDataFilePath_NoPackingSubfolderConvention reproduces a real
// Celeste "--no-packing" atlas (Misc.meta, per-chapter atlases): the .meta's
// dataFile field is just the atlas name, and each sprite's actual pixels live
// in its own file at "<atlasName>/<spritePath>.data", not "<dataFile>.data".
func TestResolveDataFilePath_NoPackingSubfolderConvention(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "Misc")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	spriteFile := filepath.Join(subdir, "purplesunset.data")
	if err := os.WriteFile(spriteFile, []byte{}, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	a := &Atlas{dir: dir}
	got, ok := a.resolveDataFilePath("Misc", "purplesunset")
	if !ok {
		t.Fatalf("expected to resolve no-packing subfolder path")
	}
	if got != spriteFile {
		t.Errorf("resolveDataFilePath = %q, want %q", got, spriteFile)
	}
}

func TestResolveDataFilePath_PackedSharedFile(t *testing.T) {
	dir := t.TempDir()
	sharedFile := filepath.Join(dir, "Gameplay0.data")
	if err := os.WriteFile(sharedFile, []byte{}, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	a := &Atlas{dir: dir}
	got, ok := a.resolveDataFilePath("Gameplay0", "tilesets/dirt")
	if !ok {
		t.Fatalf("expected to resolve packed shared file")
	}
	if got != sharedFile {
		t.Errorf("resolveDataFilePath = %q, want %q", got, sharedFile)
	}
}

func TestResolveDataFilePath_Missing(t *testing.T) {
	a := &Atlas{dir: t.TempDir()}
	if _, ok := a.resolveDataFilePath("Nope", "nope"); ok {
		t.Errorf("expected missing data file to fail to resolve")
	}
}

// TestBlendOver_CompositesOverExistingBackground guards against the
// direct-overwrite bug: a semi-transparent parallax pixel (e.g. a vignette at
// low alpha) must blend with what's already there, not replace it and wash
// out everything drawn earlier (background/tiles/decals).
func TestBlendOver_CompositesOverExistingBackground(t *testing.T) {
	dst := color.RGBA{R: 200, G: 200, B: 200, A: 255}
	src := color.RGBA64{R: 0, G: 0, B: 0, A: 0} // fully transparent source
	got := blendOver(dst, src)
	if got != dst {
		t.Errorf("fully transparent src should leave dst untouched, got %+v want %+v", got, dst)
	}

	opaqueSrc := color.RGBA64{R: 0xffff, G: 0, B: 0, A: 0xffff}
	got = blendOver(dst, opaqueSrc)
	if got.A != 255 || got.R != 255 {
		t.Errorf("fully opaque src should fully replace dst, got %+v", got)
	}
}
