package components

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/src-d/engine-cli/docker"
)

var srcdNamespaces = []string{
	"srcd",
	"bblfsh",
	"srcd-cli",
}

func List(ctx context.Context) ([]string, error) {
	c, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	imgs, err := c.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not list components: %v", err)
	}

	var res []string
	for _, img := range imgs {
		if len(img.RepoTags) == 0 {
			continue
		}

		if isSrcdComponent(img.RepoTags[0]) {
			res = append(res, img.RepoTags[0])
		}
	}

	return res, nil
}

var ErrNotSrcd = errors.New("not srcd component")

// Install installs a new component.
func Install(ctx context.Context, id string) error {
	if !isSrcdComponent(id) {
		return ErrNotSrcd
	}

	image, version := splitImageID(id)
	return docker.Pull(ctx, image, version)
}

func IsInstalled(ctx context.Context, id string) (bool, error) {
	if !isSrcdComponent(id) {
		return false, ErrNotSrcd
	}

	image, version := splitImageID(id)
	return docker.IsInstalled(ctx, image, version)
}

func splitImageID(id string) (image, version string) {
	parts := strings.Split(id, ":")
	image = parts[0]
	version = "latest"
	if len(parts) > 1 {
		version = parts[1]
	}
	return
}

func stringInSlice(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func isSrcdComponent(id string) bool {
	namespace := strings.Split(id, "/")[0]
	return stringInSlice(srcdNamespaces, namespace)
}
