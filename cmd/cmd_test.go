package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testModsDir = "../testing/Celeste/Mods"

func captureStdout(f func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		f()
		return ""
	}
	old := os.Stdout
	os.Stdout = w

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	f()

	w.Close()
	os.Stdout = old
	out := <-outC
	_ = r.Close()
	return out
}

func TestCountCollectiblesCmd(t *testing.T) {
	zipPath := filepath.Join(testModsDir, "mauve.zip")
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		t.Skipf("zip %s not found, skipping", zipPath)
	}

	out := captureStdout(func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"count-collectibles", "--mod", zipPath})
		_ = cmd.Execute()
	})

	if !strings.Contains(out, `"success":true`) {
		t.Errorf("Expected count-collectibles output to contain success:true, got: %s", out)
	}
}

func TestExportMapImagesCmd(t *testing.T) {
	zipPath := filepath.Join(testModsDir, "mauve.zip")
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		t.Skipf("zip %s not found, skipping", zipPath)
	}

	tempDir, err := os.MkdirTemp("", "export_cmd_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	out := captureStdout(func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"export-map-images", "--mod", zipPath, "--map", "smoothee/mauve/mauve", "--out", tempDir})
		_ = cmd.Execute()
	})

	if !strings.Contains(out, `"success":true`) {
		t.Errorf("Expected export-map-images output to contain success:true, got: %s", out)
	}
}
