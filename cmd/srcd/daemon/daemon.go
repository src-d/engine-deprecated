package daemon

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/blang/semver"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"

	api "github.com/src-d/engine/api"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
)

const (
	daemonImage  = "srcd/cli-daemon"
	daemonName   = "srcd-cli-daemon"
	daemonPort   = "4242"
	dockerSocket = "/var/run/docker.sock"
	workdirKey   = "WORKDIR"
)

// cli version set by src-d command
var cliVersion = ""

// SetCliVersion sets cli version
func SetCliVersion(v string) {
	cliVersion = v
}

func DockerVersion() (string, error) { return docker.Version() }
func IsRunning() (bool, error)       { return docker.IsRunning(daemonName, "") }

func Kill() error {
	cmps, err := components.List(
		context.Background(),
		true,
		components.IsWorkingDirDependant,
		components.IsRunningFilter)
	if err != nil {
		return err
	}

	for _, cmp := range cmps {
		if err := cmp.Kill(); err != nil {
			return err
		}
	}

	return docker.RemoveContainer(daemonName)
}

// Client will return a new EngineClient to interact with the daemon. If the
// daemon is not started already, it will start it at the working directory.
func Client() (api.EngineClient, error) {
	info, err := ensureStarted()
	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("0.0.0.0:%d", info.Ports[0].PublicPort)
	// TODO(campoy): add security
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return api.NewEngineClient(conn), nil
}

func Start(workdir string) error {
	_, err := start(workdir)
	return err
}

func GetLogs() (io.ReadCloser, error) {
	info, err := ensureStarted()
	if err != nil {
		return nil, err
	}

	return docker.GetLogs(context.Background(), info.ID)
}

func ensureStarted() (*docker.Container, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return start(wd)
}

func start(workdir string) (*docker.Container, error) {
	homedir, err := homedir.Dir()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get home dir")
	}

	tag, hasNew, err := GetCompatibleTag(cliVersion)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get compatible daemon version")
	}

	if hasNew {
		logrus.Warn("new version of engine is available. Please download the latest release here: https://github.com/src-d/engine/releases")
	}

	datadir := filepath.Join(homedir, ".srcd")
	if err := setupDataDirectory(workdir, datadir); err != nil {
		return nil, err
	}

	if err := docker.EnsureInstalled(daemonImage, tag); err != nil {
		return nil, err
	}

	return docker.InfoOrStart(
		context.Background(),
		daemonName,
		createDaemon(workdir, datadir, tag),
	)
}

func setupDataDirectory(workdir, datadir string) error {
	hash := sha1.Sum([]byte(workdir))
	workdirHash := hex.EncodeToString(hash[:])

	paths := [][]string{
		[]string{datadir, "gitbase", workdirHash},
	}

	for _, path := range paths {
		if err := os.MkdirAll(filepath.Join(path...), 0755); err != nil {
			return errors.Wrap(err, "unable to create data directory")
		}
	}

	return nil
}

func createDaemon(workdir, datadir, tag string) docker.StartFunc {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		config := &container.Config{
			Image:        fmt.Sprintf("%s:%s", daemonImage, tag),
			ExposedPorts: nat.PortSet{"4242": {}},
			Volumes:      map[string]struct{}{dockerSocket: {}},
			Cmd: []string{
				fmt.Sprintf("--workdir=%s", workdir),
				fmt.Sprintf("--data=%s", datadir),
			},
		}

		host := &container.HostConfig{
			PortBindings: nat.PortMap{daemonPort: {{HostPort: "4242"}}},
			Mounts: []mount.Mount{{
				Type:   mount.TypeBind,
				Source: dockerSocket,
				Target: dockerSocket,
			}},
		}

		return docker.Start(ctx, config, host, daemonName)
	}
}

// GetCompatibleTag returns semver compatible tag of daemon image by cli version
// and boolean flag if there are any newer versions with breaking changes
func GetCompatibleTag(cliVersion string) (string, bool, error) {
	if cliVersion == "" || cliVersion == "dev" {
		return "latest", false, nil
	}

	cliV, err := semver.ParseTolerant(cliVersion)
	if err != nil {
		return "", false, err
	}

	tags, err := getTags()
	if err != nil {
		return "", false, err
	}

	var breakingV semver.Version
	if cliV.Major >= 1 {
		breakingV = semver.Version{Major: cliV.Major + 1}
	} else {
		breakingV = semver.Version{Minor: cliV.Minor + 1}
	}

	var newestV semver.Version
	var hasNewBreakingTag bool
	for _, tag := range tags {
		v, err := semver.ParseTolerant(tag)
		if err != nil {
			continue
		}

		// skip pre-releases
		if len(v.Pre) > 0 {
			continue
		}

		// skip old versions
		if v.LT(cliV) {
			continue
		}

		// skip anything that breaks
		if v.GTE(breakingV) {
			hasNewBreakingTag = true
			continue
		}

		if v.GT(newestV) {
			newestV = v
		}
	}

	if newestV.Equals(semver.Version{}) {
		return "", false, errors.New("can't find compatible daemon image")
	}

	return "v" + newestV.String(), hasNewBreakingTag, nil
}

func getTags() ([]string, error) {
	c := &http.Client{}

	v := url.Values{
		"service": []string{"registry.docker.io"},
		"scope":   []string{fmt.Sprintf("repository:%s:pull", daemonImage)},
	}
	r, err := c.Get(fmt.Sprintf("https://auth.docker.io/token?%s", v.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "can't authorize in docker registry")
	}

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("incorrect status code: %d while requesting docker registry token", r.StatusCode)
	}

	var authResp struct {
		Token string
	}
	jd := json.NewDecoder(r.Body)
	err = jd.Decode(&authResp)
	if err != nil {
		return nil, errors.Wrap(err, "can't parse authorization response from docker registry")
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("https://registry-1.docker.io/v2/%s/tags/list", daemonImage), nil)
	req.Header.Add("Authorization", "Bearer "+authResp.Token)

	r, err = c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "can't request list of tags in docker registry")
	}

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("incorrect status code: %d while requesting the list of tags in docker registry", r.StatusCode)
	}

	var tagsResp struct {
		Tags []string `json:"tags"`
	}
	jd = json.NewDecoder(r.Body)
	err = jd.Decode(&tagsResp)
	if err != nil {
		return nil, errors.Wrap(err, "can't parse tags response from docker registry")
	}

	return tagsResp.Tags, nil
}
