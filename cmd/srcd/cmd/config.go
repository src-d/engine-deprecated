package cmd

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/src-d/engine/cmd/srcd/config"
	log "gopkg.in/src-d/go-log.v1"
)

// configCmd represents the config command
type configCmd struct {
	Command `name:"config" short-description:"Edit the config file" long-description:"Edit the config file.\n\nIf the config file is empty it will be populated with the default values.\nSet EDITOR env var to change the text editor command."`
}

func (c *configCmd) Execute(args []string) error {
	cmdName := os.Getenv("VISUAL")
	if cmdName == "" {
		cmdName = os.Getenv("EDITOR")
	}

	if cmdName == "" {
		if runtime.GOOS == "windows" {
			cmdName = "start" // opens the default editor
		} else {
			cmdName = "nano"
		}

		log.Debugf("EDITOR is not set, using command %s", cmdName)
	}

	configFile := c.Config
	if configFile == "" {
		var err error
		configFile, err = config.DefaultPath()
		if err != nil {
			return err
		}
	}

	emptyFile, err := isEmptyFile(configFile)
	if err != nil {
		return humanizef(err, "could not read config file contents")
	}

	if emptyFile {
		dir := filepath.Dir(configFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return humanizef(err, "can't create config directory")
		}

		err = ioutil.WriteFile(configFile, []byte(config.DefaultFileContents), 0644)
		if err != nil {
			return humanizef(err, "could not write on config file")
		}
	}

	// for Windows commands must be run using cmd /c
	cmdArgs := []string{configFile}
	if runtime.GOOS == "windows" {
		cmdArgs = append([]string{"/c", cmdName}, cmdArgs...)
		cmdName = "cmd"
	}
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	if err != nil {
		return humanizef(err, "could not launch the editor, please open %s with your preferred editor", configFile)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(&configCmd{})
}

// isEmptyFile returns true if the file does not exist or if it exists but
// contains empty text
func isEmptyFile(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}

		return true, nil
	}

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return false, err
	}

	strContents := string(contents)
	return strings.TrimSpace(strContents) == "", nil
}
