package regression

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/src-d/go-errors.v1"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-log.v1"
)

// Build structure holds information and functionality to generate
// binaries from source code.
type Build struct {
	// Version is the reference that will be built
	Version string

	// GoPath is the directory where the temporary path where the tool is built
	GoPath string

	source    string
	reference string
	url       string
	hash      string

	config Config
	tool   Tool
}

var regRepo = regexp.MustCompile(`^(local|remote|pull):([[:ascii:]]+)$`)

var (
	// ErrReferenceNotFound means that the provided reference is not found
	ErrReferenceNotFound = errors.NewKind("Reference %s not found")
	// ErrInvalidVersion means that the provided version is malformed
	ErrInvalidVersion = errors.NewKind("Version %s is invalid")
)

// IsRepo returns true if the version provided matches the repository format,
// for example: remote:master.
func IsRepo(version string) bool {
	return regRepo.MatchString(version)
}

// NewBuild creates a new Build structure
func NewBuild(
	config Config,
	tool Tool,
	version string,
) (*Build, error) {
	if !IsRepo(version) {
		return nil, ErrInvalidVersion.New(version)
	}

	source, reference := parseVersion(version)

	url := config.GitURL
	if source == "local" {
		pwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		url = fmt.Sprintf("file://%s", pwd)
	} else if url == "" {
		url = tool.GitURL
	}

	return &Build{
		Version:   version,
		source:    source,
		reference: reference,
		url:       url,

		config: config,
		tool:   tool,
	}, nil
}

// Build downloads and builds a binary from source code.
func (b *Build) Build() (string, string, error) {
	var cont bool
	var err error

	if b.Version == "local:HEAD" {
		var pwd string
		pwd, err = os.Getwd()
		if err != nil {
			return "", "", err
		}

		cont, err = b.link(pwd)
	} else {
		cont, err = b.download()
	}

	if err != nil {
		return "", "", err
	}

	defer os.RemoveAll(b.GoPath)

	// Binary is already in place, don't continue
	if !cont {
		return b.versionPath(), b.binaryPath(), nil
	}

	log.Infof("Building packages")

	for _, step := range b.tool.BuildSteps {
		err = b.buildStep(step)
		if err != nil {
			return "", "", err
		}
	}

	err = b.copyBinary()
	if err != nil {
		return "", "", err
	}

	err = b.copyExtra()
	if err != nil {
		return "", "", err
	}

	return b.versionPath(), b.binaryPath(), nil
}

func (b *Build) createProjectPath(base bool) (string, error) {
	dir, err := CreateTempDir()
	if err != nil {
		return "", err
	}

	b.GoPath = dir

	clonePath := b.projectPath()
	path := clonePath
	if !base {
		path = filepath.Dir(path)
	}

	err = os.MkdirAll(path, 0755)
	if err != nil {
		return "", err
	}

	return clonePath, nil
}

func (b *Build) link(path string) (bool, error) {
	hash, err := getCurrentCommit(path)
	if err != nil {
		return false, err
	}

	exists, err := b.checkBinary(hash)
	if err != nil || exists {
		return false, err
	}

	linkPath, err := b.createProjectPath(false)
	if err != nil {
		return false, err
	}

	err = os.Symlink(path, linkPath)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (b *Build) download() (bool, error) {
	clonePath, err := b.createProjectPath(true)
	if err != nil {
		return false, err
	}

	r, err := git.PlainInit(clonePath, false)
	if err != nil {
		return false, err
	}

	remote, err := r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{b.url},
	})

	if err != nil {
		return false, err
	}

	referenceName, hash, err := findReference(b.Version, remote)
	if err != nil {
		return false, err
	}

	exists, err := b.checkBinary(hash)
	if err != nil || exists {
		return false, err
	}

	refSpecs := []config.RefSpec{
		config.RefSpec(fmt.Sprintf("%s:refs/heads/master", referenceName)),
	}

	log.Infof("Fetching %s from %s", referenceName, b.url)

	err = r.Fetch(&git.FetchOptions{
		Depth:    1,
		RefSpecs: refSpecs,
	})
	if err != nil {
		return false, err
	}

	w, err := r.Worktree()
	if err != nil {
		return false, err
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/master"),
	})

	return true, err
}

func (b *Build) checkBinary(hash string) (bool, error) {
	b.hash = hash

	exists, err := fileExist(b.binaryPath())
	if err != nil {
		return false, err
	}

	if exists {
		log.Infof("Binary for %s (%s) already built", b.Version, hash)
	}

	return exists, nil
}

func (b *Build) buildStep(step BuildStep) error {
	cmd := exec.Command(step.Command, step.Args...)
	cmd.Dir = filepath.Join(b.projectPath(), step.Dir)
	cmd.Env = []string{
		fmt.Sprintf("GOPATH=%s", b.GoPath),
		fmt.Sprintf("PWD=%s", cmd.Dir),
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		"PKG_OS=linux",
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (b *Build) copyBinary() error {
	buildDir := filepath.Join("build", b.tool.DirName(b.config.OS))
	source := filepath.Join(b.projectPath(), buildDir, b.tool.BinName())
	destination := b.binaryPath()

	return CopyFile(source, destination, 0755)
}

func (b *Build) copyExtra() error {
	for _, e := range b.tool.ExtraFiles {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}

		s := filepath.Join(b.projectPath(), e)
		d := b.config.BinaryPath(b.hash, filepath.Base(e))

		err := CopyFile(s, d, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Build) projectPath() string {
	return filepath.Join(b.GoPath, "src", b.tool.ProjectPath)
}

func (b *Build) versionPath() string {
	return b.config.VersionPath(b.hash)
}

func (b *Build) binaryPath() string {
	return b.config.BinaryPath(b.hash, b.tool.BinName())
}

func findReference(
	version string,
	remote *git.Remote,
) (string, string, error) {
	source, reference := parseVersion(version)

	refs, err := remote.List(new(git.ListOptions))
	if err != nil {
		return "", "", err
	}

	if source == "pull" {
		name := fmt.Sprintf("refs/pull/%s/head", reference)
		for _, ref := range refs {
			if ref.Name().String() == name {
				return name, ref.Hash().String(), nil
			}
		}

		return "", "", ErrReferenceNotFound.New(reference)
	}

	for _, ref := range refs {
		name := ref.Name()

		if name.IsBranch() || name.IsTag() {
			if name.Short() == reference {
				return name.String(), ref.Hash().String(), nil
			}
		}
	}

	return "", "", ErrReferenceNotFound.New(reference)
}

func parseVersion(version string) (string, string) {
	r := regRepo.FindStringSubmatch(version)
	return r[1], r[2]
}

func getCurrentCommit(path string) (string, error) {
	r, err := git.PlainOpen(path)
	if err != nil {
		return "", err
	}

	head, err := r.Head()
	if err != nil {
		return "", err
	}

	return head.Hash().String(), nil
}
