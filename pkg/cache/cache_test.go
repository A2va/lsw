package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	tmpCache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpCache)

	tmpSource := t.TempDir()

	targetName := "hello.txt"

	content := "CONTENT"
	path := filepath.Join(tmpSource, "content.txt")
	os.WriteFile(path, []byte(content), 0644)
	url := "file://" + path

	if err := AddFile(targetName, url); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	path, err := GetFile(targetName)
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}

	checkContent(t, path, "CONTENT")

}

func TestCacheWorkflow(t *testing.T) {
	tmpCache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpCache)

	tmpSource := t.TempDir()
	targetName := "subdir/config.txt"

	// Create dummy source files
	v1Content := "VERSION 1 CONTENT"
	v1Path := filepath.Join(tmpSource, "v1.txt")
	os.WriteFile(v1Path, []byte(v1Content), 0644)
	v1URL := "file://" + v1Path

	v2Content := "VERSION 2 CONTENT"
	v2Path := filepath.Join(tmpSource, "v2.txt")
	os.WriteFile(v2Path, []byte(v2Content), 0644)
	v2URL := "file://" + v2Path

	// --- STEP 1: Initial Install ---
	t.Run("1_Install_V1", func(t *testing.T) {
		if err := AddFile(targetName, v1URL); err != nil {
			t.Fatalf("AddFile failed: %v", err)
		}

		path, err := GetFile(targetName)
		if err != nil {
			t.Fatalf("GetFile failed: %v", err)
		}

		checkContent(t, path, v1Content)
	})

	// Sleep to ensure file system timestamp changes (needed for some OSs)
	time.Sleep(100 * time.Millisecond)

	// --- STEP 2: Upgrade ---
	t.Run("2_Upgrade_To_V2", func(t *testing.T) {
		// Even though V1 exists, adding V2 should take precedence
		if err := AddFile(targetName, v2URL); err != nil {
			t.Fatalf("AddFile failed: %v", err)
		}

		path, err := GetFile(targetName)
		if err != nil {
			t.Fatalf("GetFile failed: %v", err)
		}

		checkContent(t, path, v2Content)
	})

	time.Sleep(100 * time.Millisecond)

	t.Run("3_Downgrade_To_V1", func(t *testing.T) {
		// We re-add V1. It exists in cache, but AddFile MUST 'touch' it
		// so it becomes newer than V2.
		if err := AddFile(targetName, v1URL); err != nil {
			t.Fatalf("AddFile failed: %v", err)
		}

		path, err := GetFile(targetName)
		if err != nil {
			t.Fatalf("GetFile failed: %v", err)
		}

		checkContent(t, path, v1Content)
	})
}

// Helper function to keep the main test clean
func checkContent(t *testing.T, path, expected string) {
	t.Helper() // This marks the function as a helper so logs point to the caller
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(b) != expected {
		t.Errorf("Content mismatch.\nExpected: %s\nGot:      %s", expected, string(b))
	}
}
