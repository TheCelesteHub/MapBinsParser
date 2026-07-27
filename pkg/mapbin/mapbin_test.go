package mapbin

import (
	"os"
	"path/filepath"
	"testing"
)

const testModsDir = "../../testing/Celeste/Mods"

func TestCountCollectibles(t *testing.T) {
	zipPath := filepath.Join(testModsDir, "mauve.zip")
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		t.Skipf("zip %s not found, skipping", zipPath)
	}

	res, err := CountCollectibles(zipPath)
	if err != nil {
		t.Fatalf("CountCollectibles failed: %v", err)
	}

	if !res.Success {
		t.Fatalf("CountCollectibles returned success=false: %s", res.Error)
	}

	if len(res.Maps) == 0 {
		t.Errorf("Expected at least one map in mauve.zip")
	}
}

func TestExportMapImages(t *testing.T) {
	zipPath := filepath.Join(testModsDir, "mauve.zip")
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		t.Skipf("zip %s not found, skipping", zipPath)
	}

	tempDir, err := os.MkdirTemp("", "mapbin_export_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	res, err := ExportMapImages(zipPath, "", tempDir)
	if err != nil {
		t.Fatalf("ExportMapImages failed: %v", err)
	}

	if !res.Success {
		t.Fatalf("ExportMapImages returned success=false: %s", res.Error)
	}

	manifestPath := filepath.Join(tempDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Errorf("Expected manifest.json to be created")
	}

	fullMapPath := filepath.Join(tempDir, "full_map.png")
	if _, err := os.Stat(fullMapPath); os.IsNotExist(err) {
		t.Errorf("Expected full_map.png to be created")
	}
}

func TestClassifier(t *testing.T) {
	counts := &MapCollectibleCounts{}

	CountEntity(counts, "strawberry", false, false)
	if counts.Red != 1 {
		t.Errorf("Expected Red strawberry count=1, got %d", counts.Red)
	}

	CountEntity(counts, "strawberry", true, false)
	if counts.Moon != 1 {
		t.Errorf("Expected Moon strawberry count=1, got %d", counts.Moon)
	}

	CountEntity(counts, "goldenBerry", false, false)
	if counts.Golden != 1 {
		t.Errorf("Expected Golden count=1, got %d", counts.Golden)
	}

	CountEntity(counts, "CollabUtils2/MiniHeart", false, false)
	if counts.MiniHearts != 1 {
		t.Errorf("Expected MiniHearts count=1, got %d", counts.MiniHearts)
	}

	// Trigger/controller deny rule check
	before := counts.Red
	CountEntity(counts, "strawberryTrigger", false, false)
	if counts.Red != before {
		t.Errorf("Expected trigger entity to be denied, count changed from %d to %d", before, counts.Red)
	}
}
