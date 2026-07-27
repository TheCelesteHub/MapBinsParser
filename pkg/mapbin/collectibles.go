package mapbin

import (
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func CountCollectiblesInMap(data []byte) (*MapCollectibleCounts, error) {
	r := NewBinReader(data, &MapCollectibleCounts{})
	if err := r.ReadHeader(); err != nil {
		return nil, err
	}
	if err := r.ReadElement(""); err != nil {
		return nil, err
	}
	return r.counts, nil
}

func SidFromMapPath(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	normalized = strings.TrimSuffix(normalized, filepath.Ext(normalized))
	return strings.TrimPrefix(normalized, "Maps/")
}

func IsMapBinPath(path string) bool {
	normalized := strings.ReplaceAll(path, "\\", "/")
	return strings.HasPrefix(normalized, "Maps/") && strings.HasSuffix(strings.ToLower(normalized), ".bin")
}

func CountCollectibles(modPath string) (*MapCollectiblesResult, error) {
	info, err := os.Stat(modPath)
	if err != nil {
		return nil, err
	}

	result := &MapCollectiblesResult{Success: true, Maps: map[string]*MapCollectibleCounts{}, Failed: map[string]string{}}
	if info.IsDir() {
		err = countCollectiblesInFolderMod(modPath, result)
	} else {
		err = countCollectiblesInZipMod(modPath, result)
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func countCollectiblesInZipMod(modPath string, result *MapCollectiblesResult) error {
	reader, err := zip.OpenReader(modPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if !IsMapBinPath(file.Name) {
			continue
		}
		sid := SidFromMapPath(file.Name)
		counts, err := readZipEntryMap(file)
		if err != nil {
			result.Failed[sid] = err.Error()
			continue
		}
		result.Maps[sid] = counts
	}
	return nil
}

func readZipEntryMap(file *zip.File) (*MapCollectibleCounts, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return CountCollectiblesInMap(data)
}

func countCollectiblesInFolderMod(modPath string, result *MapCollectiblesResult) error {
	mapsDir := filepath.Join(modPath, "Maps")
	if _, err := os.Stat(mapsDir); err != nil {
		return nil
	}

	return filepath.WalkDir(mapsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".bin") {
			return nil
		}
		relative, relErr := filepath.Rel(modPath, path)
		if relErr != nil {
			return nil
		}
		sid := SidFromMapPath(relative)

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			result.Failed[sid] = readErr.Error()
			return nil
		}
		counts, parseErr := CountCollectiblesInMap(data)
		if parseErr != nil {
			result.Failed[sid] = parseErr.Error()
			return nil
		}
		result.Maps[sid] = counts
		return nil
	})
}
