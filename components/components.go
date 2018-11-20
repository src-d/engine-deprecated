package components

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

func (c *Component) ImageWithVersion() string {
	return fmt.Sprintf("%s:%s", c.Image, c.Version)
}

const (
	BblfshVolume = "srcd-cli-bblfsh-storage"
)

var (
	Gitbase = Component{
		Name:    "srcd-cli-gitbase",
		Image:   "srcd/gitbase",
		Version: "v0.17.1",
	}

	GitbaseWeb = Component{
		Name:    "srcd-cli-gitbase-web",
		Image:   "srcd/gitbase-web",
		Version: "v0.3.0",
	}

	Bblfshd = Component{
		Name:    "srcd-cli-bblfshd",
		Image:   "bblfsh/bblfshd",
		Version: "v2.9.1",
	}

	BblfshWeb = Component{
		Name:    "srcd-cli-bblfsh-web",
		Image:   "bblfsh/web",
		Version: "v0.8.1",
	}

	workDirDependants = []Component{
		Gitbase,
		Bblfshd, // does not depend on workdir but it does depend on user dir
	}

	componentsList = []Component{
		Gitbase,
		GitbaseWeb,
		Bblfshd,
		BblfshWeb,
	}
)

func KnownComponents(daemonVersion string, allVersions bool) func(cmp string) bool {
	componentsList := append(componentsList, Component{
		Name:    "daemon",
		Image:   "srcd/cli-daemon",
		Version: daemonVersion,
	})

	return func(cmp string) bool {
		for _, c := range componentsList {
			var match bool
			if !allVersions {
				match = c.ImageWithVersion() == cmp
			} else {
				image, _ := splitImageID(cmp)
				match = c.Image == image
			}
			if match {
				return true
			}
		}
		return false
	}
}

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

var ErrNotSrcd = fmt.Errorf("not srcd component")

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

func Stop() error {
	logrus.Info("stopping containers...")

	// we actually not just stop but remove containers here
	// it's needed to make sure configuration of the containers is correct
	// without over-complicated logic for it
	if err := removeContainers(); err != nil {
		return errors.Wrap(err, "unable to stop all containers")
	}

	return nil
}

func Prune(images bool) error {
	logrus.Info("removing containers...")
	if err := removeContainers(); err != nil {
		return errors.Wrap(err, "unable to remove all containers")
	}

	logrus.Info("removing volumes...")

	if err := removeVolumes(); err != nil {
		return errors.Wrap(err, "unable to remove volumes")
	}

	logrus.Info("removing network...")

	if err := docker.RemoveNetwork(context.Background()); err != nil {
		return errors.Wrap(err, "unable to remove network")
	}

	if images {
		logrus.Info("removing images...")

		if err := removeImages(); err != nil {
			return errors.Wrap(err, "unable to remove all images")
		}
	}

	return nil
}

func removeContainers() error {
	cs, err := docker.List()
	if err != nil {
		return err
	}

	for _, c := range cs {
		if len(c.Names) == 0 {
			continue
		}

		name := strings.TrimLeft(c.Names[0], "/")
		if isFromEngine(name) {
			logrus.Infof("removing container %s", name)

			if err := docker.RemoveContainer(name); err != nil {
				return err
			}
		}
	}

	return nil
}

func removeVolumes() error {
	vols, err := docker.ListVolumes(context.Background())
	if err != nil {
		return err
	}

	for _, vol := range vols {
		if isFromEngine(vol.Name) {
			logrus.Infof("removing volume %s", vol.Name)

			if err := docker.RemoveVolume(context.Background(), vol.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

func removeImages() error {
	cmps, err := List(context.Background())
	if err != nil {
		return errors.Wrap(err, "unable to list images")
	}

	for _, cmp := range cmps {
		logrus.Infof("removing image %s", cmp)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		if err := docker.RemoveImage(ctx, cmp); err != nil {
			return err
		}
	}

	return nil
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

func isFromEngine(name string) bool {
	return strings.HasPrefix(name, "srcd-cli-")
}
