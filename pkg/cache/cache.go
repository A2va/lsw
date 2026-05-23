package cache

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"charm.land/log/v2"
	"github.com/hashicorp/go-getter"

	"github.com/A2va/lsw/pkg/utils"
)

// CachedFile represents a retrieved item from the cache
type CachedFile struct {
	// Path is the absolute path on disk
	// e.g. /home/user/.cache/lsw/store/subdir/nginx.tar.gz
	Path string

	// RelPath is the path relative to the downloads directory
	// e.g. subdir/nginx.tar.gz
	RelPath string

	// RealPath is the absolute path to the hashed cache artifact.
	// e.g. /home/user/.cache/lsw/store/subdir/nginx.tar.gz:a1b2c3d4e5
	RealPath string

	// RealRelPath is the hashed artifact path relative to the downloads directory.
	// e.g. subdir/nginx.tar.gz:a1b2c3d4e5
	RealRelPath string
}

var ErrFileNotFound = errors.New("file not found in cache")

var fileListCache []string
var resolvedPathCache = make(map[string]CachedFile)

// regex to identify artifacts: ends with colon + 10 hex chars
// e.g. "image.iso:a1b2c3d4e5" or "OpenSSH:a1b2c3d4e5"
var artifactReg = regexp.MustCompile(`:[0-9a-f]{10}$`)

// Name returns the stable filename on disk
// e.g. nginx.tar.gz
func (c CachedFile) Name() string {
	return filepath.Base(c.Path)
}

// Dir returns the relative directory containing the file
// e.g. subdir
func (c CachedFile) Dir() string {
	return filepath.Dir(c.RelPath)
}

// VirtualName returns the filename without the hash
// e.g. nginx.tar.gz
func (c CachedFile) VirtualName() string {
	return stripHash(c.Name())
}

// VirtualPath returns the relative path without the hash
// This is perfect for ISO structure: destination = source.VirtualPath()
// e.g. subdir/nginx.tar.gz
func (c CachedFile) VirtualPath() string {
	return filepath.Join(c.Dir(), c.VirtualName())
}

func Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	// %x converts bytes to hex string automatically
	// 5 bytes -> 10 hex chars
	return fmt.Sprintf("%x", h[:5])
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

func Add(name string, url string) error {
	if !isValidURI(url) {
		return fmt.Errorf("not a valid url")
	}

	stDir, err := getStoreDir()
	if err != nil {
		return err
	}

	log.Info("add file to cache", "name", name, "url", url)

	ext := filepath.Ext(name)
	filename := formatCacheName(filepath.Base(name), Hash(url))

	// Maintain subdirectory structure
	dst := filepath.Join(stDir, filepath.Dir(name), filename)
	linkPath := filepath.Join(stDir, name)
	log.Debug("resolved cache destination", "filename", filename, "ext", ext, "dst", dst)

	// TODO Investigate possible needed for case when switching a file to an url
	// But realistically the url will change as well
	// os.RemoveAll(dst)

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// Download if missing
	if !utils.Exists(dst) {
		if ext != "" {
			// Single File Mode
			log.Debug("download regular file")

			err := getter.GetFile(dst, url, func(c *getter.Client) error {
				c.DisableSymlinks = true
				c.Getters = getter.Getters
				c.Getters["file"] = &getter.FileGetter{Copy: true}
				return nil
			})
			if err != nil {
				return err
			}

		} else {
			// Directory/Archive Mode
			log.Debug("download archive file")
			if err := getter.Get(dst, url); err != nil {
				return err
			}

			// If we extracted a folder, check if it needs flattening
			if err := flattenSingleDirectory(dst); err != nil {
				log.Warn("error when flattening: %w", err)
			}
		}
	}

	// This ensures that even if we just downloaded an "old" file (via preservation)
	// or switched back to an existing cached file, it becomes the "active" one.
	now := time.Now()
	log.Debug("touch file to", "now", now, "file", dst)
	if err := os.Chtimes(dst, now, now); err != nil {
		return err
	}

	if err := retargetSymlink(linkPath, filepath.Base(dst)); err != nil {
		return err
	}

	// Invalidate the cache if a new file was added
	fileListCache = nil
	delete(resolvedPathCache, name)

	return nil
}

// Retrieve a file from the cache
func Get(requestedPath string) (CachedFile, error) {
	log.Info("get file in cache", "path", requestedPath)

	if item, ok := resolvedPathCache[requestedPath]; ok {
		return item, nil
	}

	stDir, err := getStoreDir()
	if err != nil {
		return CachedFile{}, err
	}

	linkPath := filepath.Join(stDir, requestedPath)
	if item, err := cachedFileFromSymlink(stDir, requestedPath, linkPath); err == nil {
		resolvedPathCache[requestedPath] = item
		return item, nil
	}

	realPath, newestTime, err := newestArtifact(stDir, requestedPath)
	if err != nil {
		return CachedFile{}, err
	}

	log.Info("found file", "path", realPath, "time", newestTime)

	if err := retargetSymlink(linkPath, filepath.Base(realPath)); err != nil {
		return CachedFile{}, err
	}

	result, err := cachedFileFromSymlink(stDir, requestedPath, linkPath)
	if err != nil {
		return CachedFile{}, err
	}

	resolvedPathCache[requestedPath] = result
	return result, nil
}

func cachedFileFromSymlink(stDir, requestedPath, linkPath string) (CachedFile, error) {
	linkInfo, err := os.Lstat(linkPath)
	if err != nil {
		return CachedFile{}, err
	}
	if linkInfo.Mode()&os.ModeSymlink == 0 {
		return CachedFile{}, fmt.Errorf("%w: %s", ErrFileNotFound, requestedPath)
	}

	info, err := os.Stat(linkPath)
	if err != nil {
		return CachedFile{}, err
	}

	realPath, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		return CachedFile{}, err
	}

	// "Touch" the winner so it isn't cleaned up by garbage collection
	now := time.Now()
	if err := os.Chtimes(realPath, now, now); err != nil {
		log.Warn("failed to touch cached file", "path", realPath, "err", err)
	}

	realRelPath, err := filepath.Rel(stDir, realPath)
	if err != nil {
		return CachedFile{}, err
	}

	log.Info("found file", "path", linkPath, "target", realPath, "time", info.ModTime())

	return CachedFile{
		Path:        linkPath,
		RelPath:     requestedPath,
		RealPath:    realPath,
		RealRelPath: realRelPath,
	}, nil
}

func newestArtifact(stDir, requestedPath string) (string, time.Time, error) {
	files, err := getFiles()
	if err != nil {
		return "", time.Time{}, err
	}

	reqDir := filepath.Dir(requestedPath)
	reqName := filepath.Base(requestedPath)

	log.Debug("resolved cache lookup", "reqDir", reqDir, "reqName", reqName)

	var newestPath string
	var newestTime time.Time
	var found bool

	for _, relPath := range files {
		if filepath.Dir(relPath) != reqDir {
			continue
		}

		if stripHash(filepath.Base(relPath)) != reqName {
			continue
		}

		absPath := filepath.Join(stDir, relPath)
		info, err := os.Stat(absPath)
		if err != nil {
			continue
		}

		if !found || info.ModTime().After(newestTime) {
			newestPath = absPath
			newestTime = info.ModTime()
			found = true
		}
	}

	if !found {
		return "", time.Time{}, fmt.Errorf("%w: %s", ErrFileNotFound, requestedPath)
	}

	return newestPath, newestTime, nil
}

func IsNotCached(err error) bool {
	if errors.Is(err, ErrFileNotFound) {
		return true
	}
	return false
}

func Init() error {
	dir, err := GetCacheDir()
	if err != nil {
		return err
	}

	dirs := []string{"store", "logs", "tmp"}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0755); err != nil {
			return err
		}
	}
	return nil
}

// helper struct to keep path and time together
type cachedFile struct {
	path    string
	modTime time.Time
}

// Prune removes old versions of files, keeping only the 'keep' most recent versions.
// accurate grouping depends on the naming convention: name.ext:hash
func Prune(keep int, maxAgeDays int) error {
	if keep < 1 {
		return fmt.Errorf("keep must be at least 1")
	}

	log.Info("prune file cache")

	files, err := getFiles()
	if err != nil {
		return err
	}

	stDir, err := getStoreDir()
	if err != nil {
		return err
	}

	groups := make(map[string][]cachedFile)

	for _, relPath := range files {
		absPath := filepath.Join(stDir, relPath)
		info, err := os.Stat(absPath)
		if err != nil {
			// If file was deleted concurrently, just skip
			continue
		}

		// Logic to reconstruct the "original" name from "name.ext:hash"
		// "subdir/image.iso:a1b2c3d4e5" -> dir: "subdir", file: "image.iso"
		dir := filepath.Dir(relPath)
		originalName := stripHash(filepath.Base(relPath))

		// Group Key: "subdir/image.iso"
		key := filepath.Join(dir, originalName)

		groups[key] = append(groups[key], cachedFile{
			path:    absPath,
			modTime: info.ModTime(),
		})
	}

	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)

	for _, versions := range groups {
		// If we don't have enough versions to prune, skip
		if len(versions) <= keep {
			continue
		}

		// Newest First
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].modTime.After(versions[j].modTime)
		})

		// Delete everything after the 'keep' index
		// e.g. if keep=1, delete from index 1 to end
		for _, fileCandidate := range versions[keep:] {
			if fileCandidate.modTime.Before(cutoff) {
				log.Debug("deleted file", "file", fileCandidate.path)
				os.RemoveAll(fileCandidate.path)
			}
		}

	}

	// Invalidate cache
	fileListCache = nil
	resolvedPathCache = make(map[string]CachedFile)

	return nil
}
