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

	"github.com/A2va/lsw/pkg/backend"
	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
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

func createBuildContext(dockerfilePath string) (io.Reader, error) {
	log.Debug("create build context")

	// Read the Dockerfile content
	body, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return nil, err
	}

	// Create a buffer to write our archive to
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	// Create a header for the "Dockerfile" file inside the tar
	header := &tar.Header{
		Name:    "Dockerfile", // The name the daemon will see
		Size:    int64(len(body)),
		Mode:    0644,
		ModTime: time.Now(),
	}

	// Write the header and the content
	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := tw.Write(body); err != nil {
		return nil, err
	}

	return buf, nil
}

type ContainerMigration struct {
	ContainerInfo       []container.InspectResponse
	OldImagesWasRemoved bool
}

// Remove old containers and images, return container info
func pruneOldVersions(c *client.Client) (ContainerMigration, error) {
	f := make(client.Filters).Add("reference", "lsw-v1:*")

	res, err := c.ImageList(context.Background(), client.ImageListOptions{All: true, Filters: f})
	if err != nil {
		return ContainerMigration{}, nil
	}

	version := config.GetVersion()
	currentTag := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)

	oldWasRemoved := false
	inspectContainers := []container.InspectResponse{}

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
				return ContainerMigration{}, err
			}

			// It should be safe to delete the container since we use volume to store data
			containers := res.Items
			for _, container := range containers {
				log.Debug("containers running on old image", "id", container.ID)

				res, err := c.ContainerInspect(context.Background(), container.ID, client.ContainerInspectOptions{})
				if err != nil {
					return ContainerMigration{}, err
				}

				inspectContainers = append(inspectContainers, res.Container)

				_, err = c.ContainerRemove(context.Background(), container.ID, client.ContainerRemoveOptions{Force: true})
				if err != nil {
					return ContainerMigration{}, err
				}

				log.Debug("prune container", "id", container.ID)
			}

			// Delete image
			oldWasRemoved = true

			log.Debug("remove old image", "id", image.ID)
			_, err = c.ImageRemove(context.Background(), image.ID, client.ImageRemoveOptions{Force: true, PruneChildren: true})
			if err != nil {
				return ContainerMigration{}, err
			}
		}
	}

	return ContainerMigration{OldImagesWasRemoved: oldWasRemoved, ContainerInfo: inspectContainers}, nil
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

	noCache := true
	remove := true
	// Use caching when developing the project to have a better iteration
	if version.Version == "dev" {
		noCache = false
		remove = false
	}

	buildOptions := client.ImageBuildOptions{
		NoCache: noCache,
		Remove:  remove,
		Tags:    []string{targetTag},
	}

	dockerfilePath, err := getDockerfile()
	if err != nil {
		log.Fatal(err)
	}

	buildContext, err := createBuildContext(dockerfilePath)
	if err != nil {
		return err
	}

	log.Debug("build image with conf:", "nocache", noCache, "remove", remove, "tag", targetTag)
	res, err := c.ImageBuild(context.Background(), buildContext, buildOptions)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	_, err = io.Copy(io.Discard, res.Body)
	if version.Version == "dev" {
		_, err = io.Copy(os.Stdout, res.Body)
	} else {
		_, err = io.Copy(io.Discard, res.Body)
	}

	return nil
}

func Init() {
	log.Info("initializing Docker provider")

	c, err := client.New(client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	migration, err := pruneOldVersions(c)
	if err != nil {
		log.Fatal(err)
	}

	err = buildImage(c)
	if err != nil {
		log.Fatal(err)
	}

	if migration.OldImagesWasRemoved {
		log.Info("restoring pruned containers")
		version := config.GetVersion()
		image := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)
		for _, container := range migration.ContainerInfo {
			// Create the same containers using the new image
			container.Config.Image = image

			createOptions := client.ContainerCreateOptions{
				Name:             container.Name,
				Config:           container.Config,
				HostConfig:       container.HostConfig,
				NetworkingConfig: &network.NetworkingConfig{EndpointsConfig: container.NetworkSettings.Networks},
			}
			c.ContainerCreate(context.Background(), createOptions)
		}
	}
}
