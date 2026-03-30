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
	"github.com/A2va/lsw/pkg/cache"
	"github.com/A2va/lsw/pkg/config"
	"github.com/A2va/lsw/pkg/utils"
	"github.com/containerd/errdefs"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/moby/client"
	"github.com/moby/term"
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
		err := cache.Add("v1/Dockerfile.v1", url)
		if err != nil {
			return "", err
		}

		item, err := cache.Get("v1/Dockerfile.v1")
		if err != nil {
			return "", err
		}
		dockerfilePath = item.Path
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

	noCache := true
	remove := true
	layers := false
	// Use caching when developing the project to have a better iteration
	if version.Version == "dev" {
		noCache = false
		remove = false
		layers = true
	}

	buildOptions := client.ImageBuildOptions{
		NoCache:     noCache,
		Remove:      remove,
		ForceRemove: remove,
		Squash:      layers,
		Tags:        []string{targetTag},
	}

	dockerfilePath, err := getDockerfile()
	if err != nil {
		utils.Panic("", err)
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
