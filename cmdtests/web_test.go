// +build integration

package cmdtests_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/src-d/engine/cmdtests"

	"github.com/stretchr/testify/suite"
)

type WebTestSuite struct {
	cmdtests.IntegrationSuite
}

func TestWebTestSuite(t *testing.T) {
	s := WebTestSuite{}
	suite.Run(t, &s)
}

func (s *WebTestSuite) testCommon(subcmd string, assertions func(url string)) {
	require := s.Require()

	r := s.StartCommand("web", []string{subcmd})
	require.NoError(r.Error, r.Combined())

	ch := make(chan error, 1)
	go func() {
		ch <- s.Wait(time.Minute, r).Error
	}()

	var url string
	exp := regexp.MustCompile(`Go to (\S+) `)

	for {
		time.Sleep(time.Second)

		if len(ch) > 0 {
			s.FailNow("Command exited unexpectedly", r.Combined())
		}

		matches := exp.FindStringSubmatch(r.Stdout())

		if matches == nil {
			continue
		}

		require.NotNil(matches)
		require.Len(matches, 2)
		url = matches[1]

		break
	}

	// Test basic GET to /
	_, err := http.Get(url)
	require.NoError(err)

	// Call any extra assertions while the web is running
	assertions(url)

	// Sending Interrupt on Windows is not implemented in go stdlib
	if runtime.GOOS == "windows" {
		// The command keeps waiting for a ctrl+c but we kill it
		err = r.Cmd.Process.Signal(os.Kill)
		require.NoError(err, r.Combined())

		// Wait for exit with error
		err = <-ch
		require.Error(err, r.Combined())
	} else {
		// The command keeps waiting for a ctrl+c
		err = r.Cmd.Process.Signal(os.Interrupt)
		require.NoError(err, r.Combined())

		// Check the exit code, from command.Wait in the goroutine
		err = <-ch
		require.NoError(err, r.Combined())
	}
}

func (s *WebTestSuite) TestSQL() {
	s.testCommon("sql", func(url string) {
		require := s.Require()

		// Call /version to verify that gitbase-web can communicate with gitbase
		// and bblfsh

		resp, err := http.Get(url + "/version")
		require.NoError(err)
		require.Equal(http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(err)

		// {"status":200,"data":{"version":"v0.6.2","bblfsh":"v2.11.8","gitbase":"8.0.11-v0.19.0"}}
		type versionResp struct {
			Status int
			Data   struct {
				Version string
				Bblfsh  string
				Gitbase string
			}
		}

		var v versionResp
		err = json.Unmarshal(body, &v)
		require.NoError(err)

		require.Equal(http.StatusOK, v.Status)

		_, err = semver.ParseTolerant(v.Data.Version)
		require.NoError(err)
		_, err = semver.ParseTolerant(v.Data.Gitbase)
		require.NoError(err)
		_, err = semver.ParseTolerant(v.Data.Bblfsh)
		require.NoError(err)
	})
}

func (s *WebTestSuite) TestParse() {
	s.testCommon("parse", func(url string) {
		require := s.Require()

		// Call /version to verify that bblfsh-web can communicate with bblfsh

		var buf = []byte("{}")
		resp, err := http.Post(url+"/api/version", "application/json", bytes.NewBuffer(buf))
		require.NoError(err)
		require.Equal(http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(err)

		// {"server":"v2.11.8","webClient":"v0.9.0"}
		type versionResp struct {
			Server    string
			WebClient string
		}

		var v versionResp
		err = json.Unmarshal(body, &v)
		require.NoError(err)

		_, err = semver.ParseTolerant(v.Server)
		require.NoError(err)
		_, err = semver.ParseTolerant(v.WebClient)
		require.NoError(err)
	})
}
