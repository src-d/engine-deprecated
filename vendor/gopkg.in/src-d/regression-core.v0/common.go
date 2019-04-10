package regression

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// CreateTempDir creates a new temporary directory in the default temp dir.
func CreateTempDir() (string, error) {
	dir, err := ioutil.TempDir("", "regression-")
	if err != nil {
		return "", err
	}

	return dir, nil
}

// RecursiveCopy copies a directory to a destination path. It creates all
// needed directories if destination path does not exist.
func RecursiveCopy(src, dst string) error {
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		err = os.MkdirAll(dst, 0700)
		if err != nil {
			return err
		}

		files, err := ioutil.ReadDir(src)
		if err != nil {
			return err
		}

		for _, file := range files {
			srcPath := filepath.Join(src, file.Name())
			dstPath := filepath.Join(dst, file.Name())

			err = RecursiveCopy(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	} else {
		err = CopyFile(src, dst, stat.Mode())
		if err != nil {
			return err
		}
	}

	return nil
}

// CopyFile makes a file copy with the specified permission.
func CopyFile(source, destination string, mode os.FileMode) error {
	exist, err := fileExist(source)
	if err != nil {
		return err
	}
	if !exist {
		return ErrBinaryNotFound.New()
	}

	orig, err := os.Open(source)
	if err != nil {
		return err
	}

	dir := filepath.Dir(destination)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}

	dst, err := os.Create(destination)
	if err != nil {
		return err
	}
	dst.Chmod(mode)
	defer dst.Close()

	_, err = io.Copy(dst, orig)
	if err != nil {
		dst.Close()
		os.Remove(dst.Name())
		return err
	}

	return nil
}
