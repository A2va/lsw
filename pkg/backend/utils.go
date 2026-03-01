package backend

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/A2va/lsw/pkg/cache"
	"github.com/A2va/lsw/pkg/config"
	"github.com/A2va/lsw/pkg/utils"
	"github.com/charmbracelet/log"
)

// https://gosamples.dev/unzip-file/
func unzipSource(source, destination string) error {
	// 1. Open the zip file
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	// 2. Get the absolute destination path
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}

	// 3. Iterate over zip files inside the archive and unzip each of them
	for _, f := range reader.File {
		err := unzipFile(f, destination)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzipFile(f *zip.File, destination string) error {
	// 4. Check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// 5. Create directory tree
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// 6. Create a destination file for unzipped content
	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}
	return nil
}

func downloadFile(url string, filepath string) error {
	// Create directory if it doesn't exists
	err := utils.CreateDir(path.Dir(filepath), 0755)
	if err != nil {
		return err
	}

	log.Info("download file", "url", url, "path", filepath)
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	if path.Ext(filepath) == ".zip" {
		unzipSource(filepath, path.Dir(filepath))
	}

	return nil
}

func DownloadFileIfNeeded(url string, file string) (string, error) {
	cachedir, err := cache.GetCacheDir()
	if err != nil {
		return "", err
	}

	filepath := path.Join(cachedir, "downloads", file)

	isZip := false

	finalFilepath := filepath

	if path.Ext(filepath) == ".zip" {
		isZip = true
		finalFilepath = strings.Replace(filepath, ".zip", "", 1)
	}

	if !utils.Exists(finalFilepath) {
		if isZip {
			filepath = finalFilepath + ".zip"
		}
		return finalFilepath, downloadFile(url, filepath)
	}

	return finalFilepath, nil
}

func CreateAllCacheDirectories() (string, error) {
	dir, err := cache.GetCacheDir()
	if err != nil {
		return "", err
	}

	err = utils.CreateDir(path.Join(dir, "downloads"), 0755)
	if err != nil {
		return "", err
	}
	err = utils.CreateDir(path.Join(dir, "iso"), 0755)
	if err != nil {
		return "", err
	}
	err = utils.CreateDir(path.Join(dir, "logs"), 0755)
	if err != nil {
		return "", err
	}
	err = utils.CreateDir(path.Join(dir, "tmp"), 0755)
	if err != nil {
		return "", err
	}

	return dir, nil
}

func GetBottle(name string) (*config.Bottle, bool) {
	if name == "" {
		return nil, false
	}

	cfg := config.Get()

	var bottleName string
	if len(name) >= 1 {
		bottleName = name
	} else {
		bottleName = cfg.DefaultBottle
	}

	log.Info("retrieving bottle", "name", bottleName)

	for i := range cfg.Bottles {
		if cfg.Bottles[i].Name == bottleName {
			log.Debug("bottle", "value", &cfg.Bottles[i], "found", true)
			return &cfg.Bottles[i], true
		}
	}

	log.Debug("bottle", "found", false)
	return nil, false
}
