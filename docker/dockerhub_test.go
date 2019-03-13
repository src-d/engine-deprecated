package docker

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var image = "srcd/cli-daemon"

func TestGetCompatibleTagUnstable(t *testing.T) {
	availableTags := []string{
		"v0.10.0",
		"v0.10.1",
		"v0.11.0-rc1",
		"v0.11.0",
		"v0.12.0-rc1",
		"v0.12.0-rc2",
	}
	dockerHubClient = newMockedClient(availableTags)

	cases := []testCase{
		// minor outdated by patch and new minor
		{
			current:        "v0.10.0",
			expected:       "v0.10.1",
			hasNewBreaking: true,
		},
		// minor latest patch and new minor
		{
			current:        "v0.10.1",
			expected:       "v0.10.1",
			hasNewBreaking: true,
		},
		// latest minor exact match
		{
			current:        "v0.11.0",
			expected:       "v0.11.0",
			hasNewBreaking: false,
		},
		// don't automatically update release candidates ever
		// minor outdated release candidad
		{
			current:        "v0.11.0-rc1",
			expected:       "v0.11.0-rc1",
			hasNewBreaking: true,
		},
		// latest minor, release candidad outdated
		{
			current:        "v0.12.0-rc1",
			expected:       "v0.12.0-rc1",
			hasNewBreaking: true,
		},
		// latest release candidad
		{
			current:        "v0.12.0-rc2",
			expected:       "v0.12.0-rc2",
			hasNewBreaking: false,
		},
	}

	testCases(t, cases)
}

func TestGetCompatibleTagStable(t *testing.T) {
	availableTags := []string{
		"v1.0.0",
		"v1.1.0-rc1",
		"v1.1.0",
		"v1.1.1",
		"v2.0.0",
		"v3.0.0-rc1",
	}
	dockerHubClient = newMockedClient(availableTags)

	cases := []testCase{
		{
			current:        "v1.0.0",
			expected:       "v1.1.1",
			hasNewBreaking: true,
		},
		{
			current:        "v1.1.0",
			expected:       "v1.1.1",
			hasNewBreaking: true,
		},
		{
			current:        "v2.0.0",
			expected:       "v2.0.0",
			hasNewBreaking: false,
		},
		{
			current:        "v1.1.0-rc1",
			expected:       "v1.1.0-rc1",
			hasNewBreaking: true,
		},
		{
			current:        "v3.0.0-rc1",
			expected:       "v3.0.0-rc1",
			hasNewBreaking: false,
		},
	}

	testCases(t, cases)
}

func TestGetCompatibleTagNotFound(t *testing.T) {
	availableTags := []string{"v1.0.0"}
	dockerHubClient = newMockedClient(availableTags)

	tag, hasNewBreaking, err := GetCompatibleTag(image, "v2.0.0")
	assert.EqualError(t, err, "can't find compatible image in docker registry for srcd/cli-daemon")
	assert.Equal(t, "", tag)
	assert.Equal(t, false, hasNewBreaking)
}

type testCase struct {
	current        string
	expected       string
	hasNewBreaking bool
}

func testCases(t *testing.T, cases []testCase) {
	for _, c := range cases {
		tag, hasNewBreaking, err := GetCompatibleTag(image, c.current)
		assert.NoError(t, err, "for tag: "+c.current)
		assert.Equal(t, c.expected, tag)
		assert.Equal(t, c.hasNewBreaking, hasNewBreaking, "for tag: "+c.current)
	}
}

func newMockedClient(tags []string) *http.Client {
	mockedT := roundTripFunc(func(req *http.Request) *http.Response {
		if req.URL.Path == "/token" {
			return newResponse(200, `{"token":"test"}`)
		}
		if req.URL.Path == "/v2/"+image+"/tags/list" {
			tString := `"` + strings.Join(tags, `","`) + `"`
			return newResponse(200, `{"tags": [`+tString+`]}`)
		}

		return newResponse(500, `{}`)
	})
	return &http.Client{Transport: mockedT}
}

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func newResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}
