package cache

import (
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	"github.com/plus3it/gorecurcopy"

	"github.com/A2va/lsw/pkg/utils"
)

func CopyFromCache(targetDir string, files []string) error {
	if err := utils.CreateDir(targetDir, 0755); err != nil {
		return err
	}

	for _, file := range files {
		item, err := Get(file)
		if err != nil {
			return err
		}

		info, err := os.Stat(item.Path)
		if err != nil {
			return err
		}

		dst := path.Join(targetDir, item.VirtualName())
		log.Debug("copy from cache", "src", item.Path, "dest", dst)

		if info.IsDir() {
			utils.CreateDir(dst, 0755)
			err = gorecurcopy.CopyDirectory(item.Path, dst)
			if err != nil {
				return err
			}
		} else {
			err = gorecurcopy.Copy(item.Path, dst)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func isValidURI(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	// Special handling for file scheme
	if u.Scheme == "file" {
		// file URIs just need a path
		return u.Path != ""
	}
	return u.Host != ""
}

func formatCacheName(filename, hash string) string {
	return filename + ":" + hash
}

func retargetSymlink(linkPath, target string) error {
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return err
	}

	tmpLink := linkPath + ".tmp-link"
	if err := os.Remove(tmpLink); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := os.Symlink(target, tmpLink); err != nil {
		return err
	}

	return os.Rename(tmpLink, linkPath)
}

// Helper to extract "image.iso" from "image.iso:a1b2c3d4e5"
func stripHash(hashedFilename string) string {
	name, _, ok := strings.Cut(hashedFilename, ":")
	if !ok {
		return hashedFilename
	}

	return name
}

func getStoreDir() (string, error) {
	root, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	stDir := path.Join(root, "store")
	log.Debug(stDir)
	return stDir, nil
}

func getFiles() ([]string, error) {
	// If cache is populated, return it immediately
	if len(fileListCache) > 0 {
		return fileListCache, nil
	}

	stDir, err := getStoreDir()
	if err != nil {
		return []string{}, err
	}

	// Walk into the cache directory
	var entries []string
	err = filepath.WalkDir(stDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Don't include the root folder itself
		if path == stDir {
			return nil
		}

		// If it matches our artifact pattern (name:hash), we treat it as an atomic item.
		// If it's a directory, we add it, but we SKIP walking inside it.
		// Detect Artifacts (Files OR Directories)
		if artifactReg.MatchString(d.Name()) {
			rel, err := filepath.Rel(stDir, path)
			if err != nil {
				return err
			}
			entries = append(entries, rel)

			// If it's a directory (extracted zip), treat it as a single unit.
			// Do NOT walk inside it.
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		return nil
	})

	if err != nil {
		return []string{}, err
	}

	return entries, nil
}

func flattenSingleDirectory(dir string) error {
	log.Info("flatten directory", "dir", dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Check if there is exactly ONE entry and it is a DIRECTORY
	if len(entries) != 1 || !entries[0].IsDir() {
		return nil // Do nothing, it's already flat or contains multiple items
	}

	singleDirName := entries[0].Name()
	singleDirPath := filepath.Join(dir, singleDirName)

	// Read the contents of that single directory
	subEntries, err := os.ReadDir(singleDirPath)
	if err != nil {
		return err
	}

	// Move every item from inside the subfolder to the root
	for _, entry := range subEntries {
		oldPath := filepath.Join(singleDirPath, entry.Name())
		newPath := filepath.Join(dir, entry.Name())

		if err := os.Rename(oldPath, newPath); err != nil {
			return err
		}
	}

	// Remove the now-empty subfolder
	return os.Remove(singleDirPath)
}
