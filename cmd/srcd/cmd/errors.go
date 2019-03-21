package cmd

import (
	"github.com/pkg/errors"
	"github.com/src-d/engine/docker"
)

// humanizef wraps and converts known errors to human friendly message
func humanizef(err error, format string, args ...interface{}) error {
	return errors.Wrapf(humanize(err), format, args...)
}

func humanize(err error) error {
	err = docker.ParseErr(err)
	errString := err.Error()

	switch e := err.(type) {
	case *docker.ContainerBindErr:
		// TODO(max): instead of placeholders we can actually ask daemon for current config if we would have such API
		confFile := "$HOME/.srcd/config.yml"
		workdir := "[workdir]"

		errString = "Port " + e.Port + " is already allocated.\n" +
			"You can define the port to be bound by " + e.Service + " in " + confFile + ", and then run:\n" +
			"srcd init " + workdir + " --config " + confFile + "\n\n" +
			"Read more in the documentation: https://docs.sourced.tech/engine/learn-more/commands#srcd"
	}

	return errors.New(errString)
}
