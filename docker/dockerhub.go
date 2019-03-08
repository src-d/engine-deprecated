package docker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/blang/semver"
	"github.com/pkg/errors"
)

// GetCompatibleTag returns the semver tag of an image compatible with the
// currentVersion, and true if there are any newer versions with breaking changes
func GetCompatibleTag(image, currentVersion string) (string, bool, error) {
	if currentVersion == "" || currentVersion == "dev" {
		return "latest", false, nil
	}

	if currentVersion == "integration-testing" {
		return currentVersion, false, nil
	}

	cliV, err := semver.ParseTolerant(currentVersion)
	if err != nil {
		return "", false, err
	}

	tags, err := getTags(image)
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
		return "", false, fmt.Errorf("can't find compatible image in docker registry for %s", image)
	}

	return "v" + newestV.String(), hasNewBreakingTag, nil
}

func getTags(image string) ([]string, error) {
	c := &http.Client{}

	v := url.Values{
		"service": []string{"registry.docker.io"},
		"scope":   []string{fmt.Sprintf("repository:%s:pull", image)},
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

	req, _ := http.NewRequest("GET", fmt.Sprintf("https://registry-1.docker.io/v2/%s/tags/list", image), nil)
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
