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
	"github.com/containers/podman/v6/libpod/define"
	"github.com/containers/podman/v6/pkg/bindings"
	"github.com/containers/podman/v6/pkg/bindings/containers"
	"github.com/containers/podman/v6/pkg/bindings/images"
	"github.com/containers/podman/v6/pkg/domain/entities/types"
	"github.com/containers/podman/v6/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func getDockerfile() (string, error) {
	log.Debug("get dockerfile")

	version := config.GetVersion()
	url := fmt.Sprintf("https://raw.githubusercontent.com/A2va/lsw/%s/Dockerfile.v1", version.Commit)

	var dockerfilePath string
	if version.Version == "dev" {
		wd, _ := os.Getwd()
		dockerfilePath = path.Join(wd, "Dockerfile.v1")
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

	log.Debug("File path", "dockerfile", dockerfilePath)

	return dockerfilePath, nil
}

type ContainerMigration struct {
	ContainerInfo       []*define.InspectContainerData
	OldImagesWasRemoved bool
}

func pruneOldVersions(c context.Context) (ContainerMigration, error) {
	f := map[string][]string{"reference": []string{"lsw-v1:*"}}
	a := true

	imagess, err := images.List(c, &images.ListOptions{
		All:     &a,
		Filters: f,
	})

	if err != nil {
		return ContainerMigration{}, err
	}

	version := config.GetVersion()
	currentTag := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)

	oldWasRemoved := false
	inspectContainers := []*define.InspectContainerData{}

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
				return ContainerMigration{}, err
			}

			for _, container := range containerss {
				log.Debug("containers runing on old image", "id", container.ID)

				res, err := containers.Inspect(c, container.ID, &containers.InspectOptions{})
				if err != nil {
					return ContainerMigration{}, err
				}

				inspectContainers = append(inspectContainers, res)

				t := true
				_, err = containers.Remove(c, container.ID, &containers.RemoveOptions{Force: &t})
				if err != nil {
					return ContainerMigration{}, err
				}

				log.Debug("prune container", "id", container.ID)
			}

			oldWasRemoved = true
			imagesToRemove = append(imagesToRemove, image.ID)
		}
	}

	log.Debug("remove old images", "id", imagesToRemove)
	t := true
	_, errs := images.Remove(c, imagesToRemove, &images.RemoveOptions{Force: &t})
	if len(errs) > 0 {
		return ContainerMigration{}, errors.Join(errs...)
	}

	return ContainerMigration{OldImagesWasRemoved: oldWasRemoved, ContainerInfo: inspectContainers}, nil

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
			log.Debug("image already exists. skipping build.")
			return nil
		}
	}

	noCache := true
	remove := true
	// Use caching when developing the project to have a better iteration
	if version.Version == "dev" {
		noCache = false
		remove = false
	}

	buildOptions := types.BuildOptions{
		BuildOptions: buildahDefine.BuildOptions{
			NoCache:                noCache,
			RemoveIntermediateCtrs: remove,
			Output:                 targetTag,
		},
	}

	dockerfilePath, err := getDockerfile()
	if err != nil {
		return err
	}

	_, err = images.Build(c, []string{dockerfilePath}, buildOptions)
	if err != nil {
		return err
	}

	return nil
}

func Init() {
	c, err := bindings.NewConnection(context.Background(), "unix:///run/podman/podman.sock")
	if err != nil {
		panic(err)
	}

	migration, err := pruneOldVersions(c)
	if err != nil {
		panic(err)
	}

	buildImage(c)

	if migration.OldImagesWasRemoved {
		version := config.GetVersion()
		image := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)
		for _, container := range migration.ContainerInfo {
			s := specgen.NewSpecGenerator(image, false)

			s.Name = container.Name + "-recreated"
			s.Env = make(map[string]string)
			for _, env := range container.Config.Env {
				// Convert "KEY=VAL" slice to map
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					s.Env[parts[0]] = parts[1]
				}
			}
			s.Command = container.Config.Cmd
			s.Entrypoint = container.Config.Entrypoint
			// s.UserNS = specgen.Namespace{
			// 	NSMode: specgen.KeepID,
			// }

			for _, mount := range container.Mounts {
				if mount.Type == "volume" {
					s.Volumes = append(s.Volumes, &specgen.NamedVolume{
						Name:    mount.Name,
						Dest:    mount.Destination,
						Options: mount.Options,
						SubPath: mount.SubPath,
					})
				} else {
					s.Mounts = append(s.Mounts, specs.Mount{
						Type:        mount.Type,
						Destination: mount.Destination,
						Source:      mount.Source,
						Options:     mount.Options,
					})
				}
			}
		}
	}

}
