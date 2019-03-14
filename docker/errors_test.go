package docker

import (
	"testing"

	"errors"

	"github.com/stretchr/testify/assert"
)

func TestErrService(t *testing.T) {
	assert := assert.New(t)

	cases := []string{
		"Error response from daemon: driver failed programming external connectivity on endpoint srcd-cli-bblfshd (9669036f7e689cb8e81d1cf81a63ea86ea42c5805d888731f5bee6cdac3cfcdf): whatever",
		"wrapped: Error response from daemon: driver failed programming external connectivity on endpoint srcd-cli-bblfshd (9669036f7e689cb8e81d1cf81a63ea86ea42c5805d888731f5bee6cdac3cfcdf): whatever",
	}

	for _, c := range cases {
		err := ParseErr(errors.New(c))
		dErr, ok := err.(*Err)
		assert.True(ok, "should return docker.Error")
		assert.Equal(dErr.Service, "srcd-cli-bblfshd")
	}
}

func TestContainerBindErr(t *testing.T) {
	assert := assert.New(t)

	e := errors.New("Error response from daemon: driver failed programming external connectivity on endpoint srcd-cli-bblfshd (9669036f7e689cb8e81d1cf81a63ea86ea42c5805d888731f5bee6cdac3cfcdf): Bind for 0.0.0.0:9432 failed: port is already allocated")
	err := ParseErr(e)
	dErr, ok := err.(*ContainerBindErr)
	assert.True(ok, "should return docker.ContainerBindErr")
	assert.Equal(dErr.Host, "0.0.0.0")
	assert.Equal(dErr.Port, "9432")
}
