package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/containerd/errdefs"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/moby/client"
	"github.com/moby/term"
	"github.com/plus3it/gorecurcopy"

	"github.com/A2va/lsw/pkg/cache"
	"github.com/A2va/lsw/pkg/config"
	"github.com/A2va/lsw/pkg/utils"
)

func copyAssetsToDir(d string) error {
	log.Debug("temp directory", "dir", d)

	version := config.GetVersion()
	if version.Version == "dev" {
		wd, _ := os.Getwd()
		gorecurcopy.CopyDirectory(path.Join(wd, "assets", "v1"), d)
	} else {
		cache.CopyFromCache(d, []string{"v1/Dockerfile.v1", "v1/wine-add-path.sh", "v1/vswhere.c"})
	}

	return nil
}

func createBuildDir() (string, error) {
	tmpDir, err := os.MkdirTemp("", "lsw-docker")
	if err != nil {
		return "", err
	}

	version := config.GetVersion()
	url := fmt.Sprintf("https://raw.githubusercontent.com/A2va/lsw/%s/assets/", version.Commit)

	if version.Version != "dev" {
		filesToCache := []string{"v1/Dockerfile.v1", "v1/vswhere.c", "v1/wine-add-apth.sh"}

		for _, file := range filesToCache {
			err := cache.Add(file, url+file)
			if err != nil {
				return "", err
			}
		}
	}

	err = copyAssetsToDir(tmpDir)
	if err != nil {
		return "", err
	}

	return tmpDir, nil
}

func createBuildContext() (io.Reader, error) {
	tmpDir, err := createBuildDir()
	if err != nil {
		return nil, err
	}

	log.Debug(tmpDir)

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	for _, file := range files {

		body, err := os.ReadFile(path.Join(tmpDir, file.Name()))
		if err != nil {
			return nil, err
		}

		// Create a header for each files inside the tar
		header := &tar.Header{
			Name:    file.Name(),
			Size:    int64(len(body)),
			Mode:    0644,
			ModTime: time.Now(),
		}

		if err := tw.WriteHeader(header); err != nil {
			return nil, err
		}

		if _, err := tw.Write(body); err != nil {
			return nil, err
		}
	}

	return buf, nil
}

// Delete running containers and remove old images
func pruneOldImages(c *client.Client) error {
	f := make(client.Filters).Add("reference", "lsw-v1:*")

	res, err := c.ImageList(context.Background(), client.ImageListOptions{All: true, Filters: f})
	if err != nil {
		return nil
	}

	version := config.GetVersion()
	currentTag := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)

	images := res.Items
	for _, image := range images {
		isOldVersion := false
		isCurrentVersion := false

		tags := image.RepoTags
		for _, tag := range tags {
			if tag == "<none>:<none>" {
				continue
			}

			if tag == currentTag {
				isCurrentVersion = true
			}

			// Check if it is an lsw image but NOT the current one
			if strings.HasPrefix(tag, "lsw-v1:") && tag != currentTag {
				isOldVersion = true
			}

		}

		if isOldVersion && !isCurrentVersion {
			log.Debug("old image found", "id", image.ID)

			f := make(client.Filters).Add("ancestor", image.ID)
			res, err := c.ContainerList(context.Background(), client.ContainerListOptions{All: true, Filters: f})
			if err != nil {
				return err
			}

			// It should be safe to delete the container since we use volume to store data
			containers := res.Items
			for _, container := range containers {
				_, err = c.ContainerRemove(context.Background(), container.ID, client.ContainerRemoveOptions{Force: true})
				if err != nil {
					return err
				}

				log.Debug("prune container", "id", container.ID)
			}

			log.Debug("remove old image", "id", image.ID)
			_, err = c.ImageRemove(context.Background(), image.ID, client.ImageRemoveOptions{Force: true, PruneChildren: true})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func buildImage(c *client.Client) error {
	version := config.GetVersion()
	targetTag := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)

	// Build the image if there isn't already one
	if version.Version != "dev" {
		_, err := c.ImageInspect(context.Background(), targetTag)
		if err == nil {
			// No error = Image exists.
			log.Info("image already exists, skipping build.")
			return nil
		}

		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to check image existence: %w", err)
		}
	}

	utils.ReportProgress("Building image", utils.ProgressStart)

	// Previously cache was disabled in non dev mode and it meant
	// that a failing build must be restarted from zero.
	// This behaviour has be changed to always enabled the cache, but prune the image left over
	// if the build was succesful.
	noCache := false
	remove := true
	// squash := true

	if version.Version == "dev" {
		noCache = false
		remove = false
		// squash = false
	}

	buildOptions := client.ImageBuildOptions{
		Dockerfile:  "Dockerfile.v1",
		NoCache:     noCache,
		Remove:      remove,
		ForceRemove: remove,
		// FIXME For now disable squash because it is experimental
		// Squash:      squash,
		Tags: []string{targetTag},
	}

	buildContext, err := createBuildContext()
	if err != nil {
		return err
	}

	log.Debug("build image with conf:", "nocache", noCache, "remove", remove, "tag", targetTag)
	res, err := c.ImageBuild(context.Background(), buildContext, buildOptions)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Display build log if in dev and debug
	if version.Version == "dev" && config.GetVersion().DebugFlag {
		termFd, isTerm := term.GetFdInfo(os.Stdout)
		err = jsonmessage.DisplayJSONMessagesStream(res.Body, os.Stdout, termFd, isTerm, nil)
		if err != nil {
			return err
		}
	} else {
		_, err = io.Copy(io.Discard, res.Body)
	}

	if version.Version != "dev" {
		utils.ReportProgress("Prune leftover images", utils.ProgressUpdate)

		filters := make(client.Filters).Add("label", "lsw-image=true")
		c.ImagePrune(context.Background(), client.ImagePruneOptions{
			Filters: filters,
		})
	}

	utils.ReportProgress("Build complete", utils.ProgressDone)

	return nil
}

func Init() {
	log.Info("initializing Docker provider")

	c, err := client.New(client.FromEnv)
	if err != nil {
		utils.Panic("", err)
	}
	defer c.Close()

	if config.GetVersion().Version != "dev" {
		err = pruneOldImages(c)
		if err != nil {
			utils.Panic("", err)
		}
	}

	err = buildImage(c)
	if err != nil {
		utils.Panic("", err)
	}
}
