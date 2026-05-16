package podman

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	buildahDefine "github.com/containers/buildah/define"
	"github.com/containers/podman/v6/pkg/bindings/containers"
	"github.com/containers/podman/v6/pkg/bindings/images"
	"github.com/containers/podman/v6/pkg/domain/entities/types"
	"github.com/plus3it/gorecurcopy"

	"github.com/A2va/lsw/pkg/cache"
	"github.com/A2va/lsw/pkg/config"
	"github.com/A2va/lsw/pkg/utils"
)

func getDockerfile() string {
	_, exist := os.LookupEnv("_LSW_CI")
	if exist {
		return "v1/Dockerfile.ci"
	}
	return "v1/Dockerfile.v1"
}

func copyBuildAssetsToDir(d string) error {
	log.Debug("temp directory", "dir", d)

	version := config.GetVersion()
	if version.Version == "dev" {
		wd, _ := os.Getwd()
		gorecurcopy.CopyDirectory(path.Join(wd, "assets", "v1"), d)
	} else {
		cache.CopyFromCache(d, []string{getDockerfile(), "v1/wine-add-path.sh", "v1/vswhere.c", "v1/setup-msvc.sh"})
	}

	return nil
}

// Delete running containers and remove old images
func pruneOldImages(c context.Context) error {
	f := map[string][]string{"reference": []string{"lsw-v1:*"}}
	a := true

	imagess, err := images.List(c, &images.ListOptions{
		All:     &a,
		Filters: f,
	})

	if err != nil {
		return err
	}

	version := config.GetVersion()
	currentTag := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)

	imagesToRemove := []string{}

	for _, image := range imagess {

		isOldVersion := false
		isCurrentVersion := false

		tags := image.RepoTags
		for _, fullTag := range tags {
			if fullTag == "<none>:<none>" {
				continue
			}

			// Remove the registry prefix (e.g., "localhost/lsw-v1:abc" -> "lsw-v1:abc")
			normalizedTag := fullTag
			if lastSlash := strings.LastIndex(fullTag, "/"); lastSlash != -1 {
				normalizedTag = fullTag[lastSlash+1:]
			}

			// Check if it is an lsw image but NOT the current one
			if normalizedTag == currentTag {
				isCurrentVersion = true
			} else if strings.HasPrefix(normalizedTag, "lsw-v1:") {
				isOldVersion = true
			}
		}

		if isOldVersion && !isCurrentVersion {
			log.Debug("old image found", "id", image.ID)

			f := map[string][]string{"ancestor": []string{"lsw-v1:*"}}
			containerss, err := containers.List(c, &containers.ListOptions{Filters: f})
			if err != nil {
				return err
			}

			for _, container := range containerss {
				log.Debug("containers running on old image", "id", container.ID)

				t := true
				_, err = containers.Remove(c, container.ID, &containers.RemoveOptions{Force: &t})
				if err != nil {
					return err
				}

				log.Debug("prune container", "id", container.ID)
			}

			imagesToRemove = append(imagesToRemove, image.ID)
		}
	}

	if len(imagesToRemove) > 0 {
		log.Debug("remove old images", "id", imagesToRemove)
		t := true
		fa := false
		_, errs := images.Remove(c, imagesToRemove, &images.RemoveOptions{Force: &t, NoPrune: &fa})
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
	}

	return nil

}

func createBuildDir() (string, error) {
	tmpDir, err := os.MkdirTemp("", "lsw-podman")
	if err != nil {
		return "", err
	}

	version := config.GetVersion()
	url := fmt.Sprintf("https://raw.githubusercontent.com/A2va/lsw/%s/assets/", version.Commit)

	if version.Version != "dev" {
		filesToCache := []string{getDockerfile(), "v1/vswhere.c", "v1/wine-add-path.sh", "v1/setup-msvc.sh"}

		for _, file := range filesToCache {
			err := cache.Add(file, url+file)
			if err != nil {
				return "", err
			}
		}
	}

	err = copyBuildAssetsToDir(tmpDir)
	if err != nil {
		return "", err
	}

	return tmpDir, nil
}

func buildImage(c context.Context) error {
	version := config.GetVersion()
	targetTag := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)

	// Build the image if there isn't already one
	if version.Version != "dev" {
		exist, err := images.Exists(c, targetTag, &images.ExistsOptions{})
		if err != nil {
			return err
		}

		if exist {
			log.Info("image already exists, skipping build.")
			return nil
		}
	}

	utils.ReportProgress("Building image", utils.ProgressStart)

	// Previously cache was disabled in non dev mode and it meant
	// that a failing build must be restarted from zero.
	// This behaviour has be changed to always enabled the cache, but prune the image left over
	// if the build was succesful.
	noCache := false
	layers := true
	squash := true

	if version.Version == "dev" {
		noCache = false
		layers = true
		squash = false
	}

	buildDir, err := createBuildDir()
	if err != nil {
		return err
	}

	var outWriter io.Writer = io.Discard
	var errWriter io.Writer = io.Discard

	// Display build log if in dev and debug
	if version.Version == "dev" && config.GetVersion().DebugFlag {
		outWriter = os.Stdout
		errWriter = os.Stderr
	}

	buildOptions := types.BuildOptions{
		BuildOptions: buildahDefine.BuildOptions{
			ContextDirectory:        buildDir,
			NoCache:                 noCache,
			RemoveIntermediateCtrs:  true,
			ForceRmIntermediateCtrs: true,
			Layers:                  layers,
			Labels:                  []string{"lsw-image=true"},
			Squash:                  squash,
			Output:                  targetTag,
			PullPolicy:              buildahDefine.PullIfMissing,
			// Cache only works if the ouput format are defined
			OutputFormat: buildahDefine.OCIv1ImageManifest,
			// OutputFormat: buildahDefine.Dockerv2ImageManifest,
			Out:          outWriter,
			Err:          errWriter,
			ReportWriter: outWriter,
		},
	}
	log.Debug("build options", "opts", buildOptions)

	_, err = images.Build(c, []string{filepath.Base(getDockerfile())}, buildOptions)
	if err != nil {
		return err
	}

	if version.Version != "dev" {
		utils.ReportProgress("Prune leftover images", utils.ProgressUpdate)

		t := true
		filters := map[string][]string{
			"label": {"lsw-image=true"},
		}
		pruneReport, err := images.Prune(c, &images.PruneOptions{
			External:   &t,
			BuildCache: &t,
			Filters:    filters,
		})
		if err != nil {
			log.Warn("Failed to clean up build cache, but image was built successfully", "err", err)
		} else {
			log.Debug("Cleaned up dangling cache", "deleted", len(pruneReport))
		}
	}

	utils.ReportProgress("Build complete", utils.ProgressDone)
	return nil
}

func Init() {
	log.Info("initializing Podman provider")

	c, err := podmanClient()
	if err != nil {
		utils.Panic("", err)
	}

	// Prune old image only in dev version, to rely on the podman cache
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
