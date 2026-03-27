package docker

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"charm.land/log/v2"
	"github.com/A2va/lsw/pkg/backend"
	"github.com/A2va/lsw/pkg/config"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
)

func hash() string {
	t := time.Now().String()
	bs := sha1.Sum([]byte(t))
	return hex.EncodeToString(bs[:])[:7]
}

func createOptions(bottle config.Bottle) (client.ContainerCreateOptions, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return client.ContainerCreateOptions{}, err
	}

	version := config.GetVersion()
	image := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)
	volumeName := fmt.Sprintf("lsw-v1-%s", bottle.Name)

	var mounts []mount.Mount
	mounts = append(mounts, mount.Mount{
		Type:   mount.TypeVolume,
		Source: volumeName,
		Target: "/opt/prefix",
	})

	mounts = append(mounts, mount.Mount{
		Type:   mount.TypeBind,
		Source: cwd,
		Target: cwd,
	})

	for _, m := range bottle.Mounts {
		mountPath, err := filepath.Abs(m)
		if err != nil {
			return client.ContainerCreateOptions{}, err
		}
		mounts = append(mounts, mount.Mount{Type: mount.TypeBind, Source: mountPath, Target: mountPath})
	}

	userStr := fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())
	log.Debug("user string", "userStr", userStr)

	name := fmt.Sprintf("lsw-%s-%s", bottle.Name, hash())
	createOpts := client.ContainerCreateOptions{
		Name: name,
		Config: &container.Config{
			Image: image,
			Cmd:   []string{"wine", backend.GetShell(bottle)},
			Env: []string{
				"HOME=/opt/prefix", // Points HOME to the writable volume
			},

			// Cmd:          []string{"bash"},
			Tty:          true,
			OpenStdin:    true,
			AttachStdin:  true,
			AttachStdout: true,
			WorkingDir:   cwd,
			User:         userStr,
		},
		HostConfig: &container.HostConfig{
			Mounts: mounts,
		},
	}

	log.Debug("docker provider", "createOptions", createOpts)
	return createOpts, nil
}
