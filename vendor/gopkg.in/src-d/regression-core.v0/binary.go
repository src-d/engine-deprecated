package regression

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alcortesm/tgz"
	"gopkg.in/src-d/go-errors.v1"
	"gopkg.in/src-d/go-log.v1"
)

var regRelease = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// ErrBinaryNotFound is returned when the executable is not found in
// the release tarball.
var ErrBinaryNotFound = errors.NewKind("binary not found in release tarball")

// ErrTarball is returned when there is a problem with a tarball.
var ErrTarball = errors.NewKind("cannot unpack tarball %s")

// ErrExtraFile is returned when there is a problem saving an extra file.
var ErrExtraFile = errors.NewKind("cannot save extra file %s")

// Binary struct contains information and functionality to prepare and
// use a binary version.
type Binary struct {
	Version string
	Path    string

	releases *Releases
	config   Config
	tool     Tool
	cacheDir string
}

// NewBinary creates a new Binary structure.
func NewBinary(
	config Config,
	tool Tool,
	version string,
	releases *Releases,
) *Binary {
	return &Binary{
		Version:  version,
		releases: releases,
		config:   config,
		tool:     tool,
	}
}

// IsRelease checks if the version matches the format of a release, for
// example v0.12.1.
func (b *Binary) IsRelease() bool {
	return regRelease.MatchString(b.Version)
}

// Download prepares a binary version if it's still not in the
// binaries directory.
func (b *Binary) Download() error {
	switch {
	case IsRepo(b.Version):
		build, err := NewBuild(b.config, b.tool, b.Version)
		if err != nil {
			return err
		}

		cacheDir, binary, err := build.Build()
		if err != nil {
			return err
		}

		b.Path = binary
		b.cacheDir = cacheDir
		return nil

	case b.Version == "latest":
		version, err := b.releases.Latest()
		if err != nil {
			return nil
		}

		b.Version = version

	case !b.IsRelease():
		b.Path = b.Version
		b.cacheDir = filepath.Dir(b.Version)
		return nil
	}

	cacheName := b.cacheName()
	exist, err := fileExist(cacheName)
	if err != nil {
		return err
	}

	b.Path = cacheName
	b.cacheDir = filepath.Dir(cacheName)

	if exist {
		log.Debugf("Binary for %s already downloaded", b.Version)
		return nil
	}

	log.Debugf("Downloading version %s", b.Version)
	err = b.downloadRelease()
	if err != nil {
		log.Errorf(err, "Could not download version %s", b.Version)
		return err
	}

	return nil
}

// ExtraFile returns the path from a file in the cache directory.
func (b *Binary) ExtraFile(name string) string {
	return filepath.Join(b.cacheDir, name)
}

func (b *Binary) downloadRelease() error {
	tmpDir, err := CreateTempDir()
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	download := filepath.Join(tmpDir, "download.tar.gz")
	source, err := b.releases.Get(b.Version, b.tarName(), download)
	if err != nil {
		return err
	}

	path, err := tgz.Extract(download)
	if err != nil {
		return err
	}
	defer os.RemoveAll(path)

	binary := filepath.Join(path, b.dirName(), b.tool.BinName())
	err = CopyFile(binary, b.cacheName(), 0755)
	if err != nil {
		return err
	}

	if len(b.tool.ExtraFiles) > 0 {
		extra := filepath.Join(tmpDir, "extra.tar.gz")
		err := Download(source, extra)
		if err != nil {
			return ErrTarball.Wrap(err, extra)
		}

		// skip the first directory as files inside the source tar are inside
		// a directory, for example:
		//   src-d-gitbase-c533882/_testdata/regression.yml
		cachePath := b.config.VersionPath(b.Version)
		err = GetExtras(extra, cachePath, b.tool.ExtraFiles, 1)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Binary) cacheName() string {
	return b.config.BinaryPath(b.Version, b.tool.BinName())
}

func (b *Binary) tarName() string {
	return fmt.Sprintf("%s_%s_%s_amd64.tar.gz",
		b.tool.Name,
		b.Version,
		b.config.OS,
	)
}

func (b *Binary) dirName() string {
	return b.tool.DirName(b.config.OS)
}

func fileExist(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// GetExtras extracts the files from the tarball contained in the extras list
// to the provided path. It can skip directories with the parameter depth.
func GetExtras(tarball, path string, extras []string, depth int) error {
	r, err := os.Open(tarball)
	if err != nil {
		return ErrTarball.Wrap(err, tarball)
	}
	defer r.Close()

	gr, err := gzip.NewReader(r)
	if err != nil {
		return ErrTarball.Wrap(err, tarball)
	}

	t := tar.NewReader(gr)

	for {
		h, err := t.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return ErrTarball.Wrap(err, tarball)
		}

		name := skipDir(h.Name, depth)
		if contains(extras, name) {
			p := filepath.Join(path, filepath.Base(name))
			err = IOToFile(t, p)
			if err != nil {
				return ErrExtraFile.Wrap(err, p)
			}
		}
	}
}

func skipDir(name string, depth int) string {
	if depth < 1 {
		return name
	}

	s := strings.SplitN(name, string(os.PathSeparator), depth+1)
	if len(s) < depth+1 {
		return ""
	}

	return s[depth]
}

func contains(items []string, name string) bool {
	for _, s := range items {
		if s == name {
			return true
		}
	}

	return false
}
