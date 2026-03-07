package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	// Reset the internal cache
	fileListCache = nil
	resolvedPathCache = make(map[string]string)

	tmpCache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpCache)

	fmt.Println(tmpCache)

	tmpSource := t.TempDir()

	targetName := "hello.txt"

	content := "CONTENT"
	path := filepath.Join(tmpSource, "content.txt")
	os.WriteFile(path, []byte(content), 0644)
	url := "file://" + path

	if err := Add(targetName, url); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	path, err := Get(targetName)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	checkContent(t, path, "CONTENT")
}

func TestPrune(t *testing.T) {
	// Reset globals
	fileListCache = nil
	resolvedPathCache = make(map[string]string)

	tmpCache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpCache)

	tmpSource := t.TempDir()
	targetName := "images/ubuntu.iso"

	// Create 3 dummy files with distinct content
	// We sleep to ensure timestamps differ
	for i := range 3 {
		content := fmt.Sprintf("CONTENT v%d", i)
		src := filepath.Join(tmpSource, fmt.Sprintf("v%d.iso", i))
		os.WriteFile(src, []byte(content), 0644)

		if err := Add(targetName, "file://"+src); err != nil {
			t.Fatalf("Failed to add v%d: %v", i, err)
		}
		time.Sleep(50 * time.Millisecond) // Ensure ModTime differs
	}

	// Verify we have 3 files in the specific directory
	files, _ := getFiles()
	if len(files) != 3 {
		t.Fatalf("Expected 3 files before prune, got %d", len(files))
	}

	// Keep only the 1 newest file
	if err := Prune(1, 0); err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	files, _ = getFiles()
	if len(files) != 1 {
		t.Errorf("Expected 1 file after prune, got %d", len(files))
	}

	// Verify the remaining file is actually the newest (V2)
	path, err := Get(targetName)
	if err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(path)
	if string(content) != "CONTENT v2" {
		t.Errorf("Prune deleted the wrong file! Remaining content: %s", string(content))
	}
}

func TestCacheWorkflow(t *testing.T) {
	// Reset the internal cache
	fileListCache = nil
	resolvedPathCache = make(map[string]string)

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
		if err := Add(targetName, v1URL); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		path, err := Get(targetName)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		checkContent(t, path, v1Content)
	})

	// Sleep to ensure file system timestamp changes (needed for some OSs)
	time.Sleep(100 * time.Millisecond)

	// --- STEP 2: Upgrade ---
	t.Run("2_Upgrade_To_V2", func(t *testing.T) {
		// Even though V1 exists, adding V2 should take precedence
		if err := Add(targetName, v2URL); err != nil {
			t.Fatalf("AddFile failed: %v", err)
		}

		path, err := Get(targetName)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		checkContent(t, path, v2Content)
	})

	time.Sleep(100 * time.Millisecond)

	t.Run("3_Downgrade_To_V1", func(t *testing.T) {
		// We re-add V1. It exists in cache, but AddFile MUST 'touch' it
		// so it becomes newer than V2.
		if err := Add(targetName, v1URL); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		path, err := Get(targetName)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		checkContent(t, path, v1Content)
	})
}

func TestAddFile_Archive(t *testing.T) {
	// Reset Cache
	fileListCache = nil
	resolvedPathCache = make(map[string]string)

	tmpCache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpCache)

	const version = "v8.1.0.0p1-Beta"
	url := "https://github.com/PowerShell/Win32-OpenSSH/releases/download/" + version + "/OpenSSH-Win64.zip"

	// Request: Store in "OpenSSH" (implied directory, no extension)
	targetName := "OpenSSH"

	if err := Add(targetName, url); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Retrieve
	path, err := Get(targetName)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Verification
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	// Should be a directory (because it was a zip)
	if !info.IsDir() {
		t.Errorf("Expected path to be a directory (extracted zip), got file: %s", path)
	}

	// Should contain specific files from that zip
	expectedFile := filepath.Join(path, "ssh.exe") // The zip contains a root folder OpenSSH-Win64
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Did not find expected content inside zip: %s", expectedFile)
	}

	// Check Directory Name Format
	// Expected: OpenSSH-<hash>
	base := filepath.Base(path)
	if len(base) < 12 || base[:8] != "OpenSSH-" { // OpenSSH- + 10 chars hash
		t.Errorf("Directory name format wrong. Expected OpenSSH-HASH..., got: %s", base)
	}
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
