package components

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/src-d/engine/docker"
)

var srcdNamespaces = []string{
	"srcd",
	"bblfsh",
}

type Component struct {
	Name    string
	Image   string
	Version string // only if there's a required version
}

var (
	Gitbase = Component{
		Name:  "srcd-cli-gitbase",
		Image: "srcd/gitbase",
	}

	GitbaseWeb = Component{
		Name:  "srcd-cli-gitbase-web",
		Image: "srcd/gitbase-web",
	}

	Bblfshd = Component{
		Name:  "srcd-cli-bblfshd",
		Image: "bblfsh/bblfshd",
	}

	BblfshWeb = Component{
		Name:  "srcd-cli-bblfsh-web",
		Image: "bblfsh/web",
	}

	Pilosa = Component{
		Name:    "srcd-cli-pilosa",
		Image:   "pilosa/pilosa",
		Version: "v0.9.0",
	}

	workDirDependants = []Component{
		Gitbase,
		Pilosa,
		Bblfshd, // does not depend on workdir but it does depend on user dir
	}
)

type FilterFunc func(string) bool

func filter(cmps []string, filters []FilterFunc) []string {
	var result []string
	for _, cmp := range cmps {
		var add = true
		for _, f := range filters {
			if !f(cmp) {
				add = false
				break
			}
		}

		if add {
			result = append(result, cmp)
		}
	}
	return result
}

func IsWorkingDirDependant(cmp string) bool {
	for _, c := range workDirDependants {
		if c.Name == cmp {
			return true
		}
	}
	return false
}

func List(ctx context.Context, filters ...FilterFunc) ([]string, error) {
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

	if len(filters) > 0 {
		return filter(res, filters), nil
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
