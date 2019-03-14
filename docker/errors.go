package docker

import (
	"regexp"
	"strings"
)

// Err is a basic docker errors to provide more context to the client
//
// docker API doesn't export internal errors so we parse error message to distinguish them
//
// example of an error:
// https://github.com/docker/libnetwork/blob/b0186632522c68f4e1222c4f6d7dbe518882024f/endpoint.go#L583
// implementation:
// https://github.com/docker/libnetwork/blob/dcb8d9b31a7449f908e6efc2f9d5ab8dc6adefb1/types/types.go#L633
type Err struct {
	Service string
	Err     error
}

// ContainerBindErr happens when container can't be bind
type ContainerBindErr struct {
	*Err
	Host string
	Port string
}

// ParseErr parses error message and converts error to docker error if possible
func ParseErr(err error) error {
	if !strings.Contains(err.Error(), "Error response from daemon: ") {
		return err
	}

	dErr := &Err{
		Service: getServiceName(err),
		Err:     err,
	}

	if ok, dErr := parseContainerBindError(dErr); ok {
		return dErr
	}

	return dErr
}

// Error implements error interface
func (e *Err) Error() string {
	return e.Err.Error()
}

var regexpOnEndpoint = regexp.MustCompile(` on endpoint (\S+)`)

// currently support only errors from libnetwork/endpoint
// there might be some others potentionally
func getServiceName(err error) string {
	m := regexpOnEndpoint.FindStringSubmatch(err.Error())
	if len(m) > 0 {
		return m[1]
	}

	return ""
}

var regexpBind = regexp.MustCompile(`Bind for (\d{1,3}\.\d{1,3}.\d{1,3}.\d{1,3}):(\d+) failed: port is already allocated`)

// currently support only "port is already allocated"
func parseContainerBindError(dErr *Err) (bool, error) {
	m := regexpBind.FindStringSubmatch(dErr.Err.Error())
	if len(m) > 0 {
		return true, &ContainerBindErr{
			Err:  dErr,
			Host: m[1],
			Port: m[2],
		}
	}

	return false, dErr
}
