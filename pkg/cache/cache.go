package cache

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-getter"
)

func hash(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:5]
}

func Exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func GetCacheDir() (string, error) {
	c, exist := os.LookupEnv("XDG_CACHE_HOME")

	if exist {
		return filepath.Join(c, "lsw"), nil
	}

	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return "", homeErr
	}

	return filepath.Join(home, ".cache", "lsw"), nil
}

func AddFile(name string, url string) error {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(filepath.Base(name), ext)
	filename := fmt.Sprintf("%s-%s%s", base, hash(url), ext)

	// Maintain subdirectory structure
	dst := filepath.Join(cacheDir, filepath.Dir(name), filename)

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// Download if missing
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		if err := getter.GetFile(dst, url); err != nil {
			return err
		}
	}

	// This ensures that even if we just downloaded an "old" file (via preservation)
	// or switched back to an existing cached file, it becomes the "active" one.
	now := time.Now()
	return os.Chtimes(dst, now, now)
}

// Retrieve a file from the cache
func GetFile(requestedPath string) (string, error) {
	// FIXME cache the list of files, invalidate when calling AddFile
	// Get list of all files in cache
	files, err := getFiles()
	if err != nil {
		return "", err
	}

	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}

	// Parse the input path (e.g. "subdir/file.txt")
	reqDir := filepath.Dir(requestedPath)
	reqExt := filepath.Ext(requestedPath)
	reqBase := strings.TrimSuffix(filepath.Base(requestedPath), reqExt)

	// We look for files starting with "file-" to account for the hash suffix
	reqPrefix := reqBase + "-"

	var newestPath string
	var newestTime time.Time
	var found bool

	for _, relPath := range files {
		// Filter by Directory
		if filepath.Dir(relPath) != reqDir {
			continue
		}

		// Filter by Extension
		if filepath.Ext(relPath) != reqExt {
			continue
		}

		// Filter by Filename prefix
		baseName := filepath.Base(relPath)
		if !strings.HasPrefix(baseName, reqPrefix) {
			continue
		}

		// Check the file stats
		absPath := filepath.Join(cacheDir, relPath)
		info, err := os.Stat(absPath)
		if err != nil {
			continue
		}

		// Update the tracker if this file is newer
		if !found || info.ModTime().After(newestTime) {
			newestPath = absPath
			newestTime = info.ModTime()
			found = true
		}
	}

	if !found {
		return "", fmt.Errorf("file not found in cache: %s", requestedPath)
	}

	// "Touch" the winner so it isn't cleaned up by garbage collection
	now := time.Now()
	if err := os.Chtimes(newestPath, now, now); err != nil {
		// Even if we fail to touch it (permissions?), we should still return the file
		// Log error if you have a logger
	}
	return newestPath, nil
}

func getFiles() ([]string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return []string{}, err
	}

	// Walk into the cache directory
	var entries []string
	err = filepath.WalkDir(cacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(cacheDir, path)
		if err != nil {
			return err
		}

		if !d.IsDir() {
			entries = append(entries, rel)
		}

		return nil
	})

	if err != nil {
		return []string{}, err
	}

	return entries, nil
}
