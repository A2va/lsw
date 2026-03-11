package podman

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/A2va/lsw/pkg/backend"
	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	buildahDefine "github.com/containers/buildah/define"

	"github.com/containers/podman/v6/pkg/bindings/containers"
	"github.com/containers/podman/v6/pkg/bindings/images"
	"github.com/containers/podman/v6/pkg/domain/entities/types"
)

func getDockerfile() (string, error) {
	log.Debug("get dockerfile")

	version := config.GetVersion()
	url := fmt.Sprintf("https://raw.githubusercontent.com/A2va/lsw/%s/assets/v1/Dockerfile", version.Commit)

	var dockerfilePath string
	if version.Version == "dev" {
		wd, _ := os.Getwd()
		dockerfilePath = path.Join(wd, "assets", "v1", "Dockerfile")
	} else {
		_, err := backend.DownloadFileIfNeeded(url, "Dockerfile.v1")
		if err != nil {
			return "", err
		}

		cache, err := backend.GetCacheDir()
		if err != nil {
			return "", err
		}
		dockerfilePath = path.Join(cache, "downloads", "Dockerfile.v1")
	}

	log.Debug("file path", "dockerfile", dockerfilePath)

	return dockerfilePath, nil
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

	log.Debug("remove old images", "id", imagesToRemove)
	t := true
	_, errs := images.Remove(c, imagesToRemove, &images.RemoveOptions{Force: &t})
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil

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

	noCache := true
	remove := true
	layers := false

	// Use caching when developing the project to have a better iteration
	if version.Version == "dev" {
		noCache = false
		remove = false
		layers = true
	}

	dockerfilePath, err := getDockerfile()
	if err != nil {
		log.Fatal(err)
	}

	contextDir := path.Dir(dockerfilePath)
	dockerfileName := path.Base(dockerfilePath)

	buildOptions := types.BuildOptions{
		BuildOptions: buildahDefine.BuildOptions{
			ContextDirectory:        contextDir,
			NoCache:                 noCache,
			RemoveIntermediateCtrs:  remove,
			ForceRmIntermediateCtrs: remove,
			Layers:                  layers,
			Output:                  targetTag,
			PullPolicy:              buildahDefine.PullIfMissing,
			// Cache only works if the ouput format are defined
			OutputFormat: buildahDefine.OCIv1ImageManifest,
			// OutputFormat: buildahDefine.Dockerv2ImageManifest,
		},
	}
	log.Debug("build options", "opts", buildOptions)

	_, err = images.Build(c, []string{dockerfileName}, buildOptions)
	if err != nil {
		log.Fatalf("err: %w", err)
	}

	return nil
}

func Init() {
	log.Info("initializing Podman provider")

	c, err := podmanClient()
	if err != nil {
		log.Fatal(err)
	}

	// Prune old image only in dev version, to rely on the podman cache
	if config.GetVersion().Version != "dev" {
		err = pruneOldImages(c)
		if err != nil {
			log.Fatal(err)
		}
	}

	buildImage(c)
}
