package mapbin

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func buildFixturePNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	buf := &bytes.Buffer{}
	_ = png.Encode(buf, img)
	return buf.Bytes()
}

func TestModAssetResolver_Folder(t *testing.T) {
	dir := t.TempDir()
	pngDir := filepath.Join(dir, "Graphics", "Atlases", "Gameplay", "bgs", "author")
	if err := os.MkdirAll(pngDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pngDir, "bg.png"), buildFixturePNG(), 0644); err != nil {
		t.Fatalf("write fixture png: %v", err)
	}

	resolver, closeFn := newModAssetResolver(dir)
	if resolver == nil {
		t.Fatalf("expected resolver for folder mod")
	}
	if closeFn != nil {
		closeFn()
	}

	img, ok := resolver.getSprite("bgs/author/bg")
	if !ok {
		t.Fatalf("expected bgs/author/bg to resolve")
	}
	if b := img.Bounds(); b.Dx() != 2 || b.Dy() != 2 {
		t.Errorf("expected 2x2 sprite, got %dx%d", b.Dx(), b.Dy())
	}

	if _, ok := resolver.getSprite("bgs/author/missing"); ok {
		t.Errorf("expected missing sprite to miss")
	}
}

func TestModAssetResolver_Zip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "mod.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("Graphics/Atlases/Gameplay/decals/author/deco.png")
	if err != nil {
		t.Fatalf("zip create entry: %v", err)
	}
	if _, err := w.Write(buildFixturePNG()); err != nil {
		t.Fatalf("zip write entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	f.Close()

	resolver, closeFn := newModAssetResolver(zipPath)
	if resolver == nil {
		t.Fatalf("expected resolver for zip mod")
	}
	if closeFn != nil {
		defer closeFn()
	}

	img, ok := resolver.getSprite("decals/author/deco")
	if !ok {
		t.Fatalf("expected decals/author/deco to resolve")
	}
	if b := img.Bounds(); b.Dx() != 2 || b.Dy() != 2 {
		t.Errorf("expected 2x2 sprite, got %dx%d", b.Dx(), b.Dy())
	}

	if _, ok := resolver.getSprite("decals/author/missing"); ok {
		t.Errorf("expected missing sprite to miss")
	}
}

func TestNewModAssetResolver_NonDirNonZip(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "map.bin")
	if err := os.WriteFile(binPath, []byte{0}, 0644); err != nil {
		t.Fatalf("write fixture bin: %v", err)
	}

	resolver, closeFn := newModAssetResolver(binPath)
	if resolver != nil {
		t.Errorf("expected nil resolver for bare .bin mod path")
	}
	if closeFn != nil {
		t.Errorf("expected nil closer for bare .bin mod path")
	}
}
