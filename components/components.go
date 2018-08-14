package components

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func List(ctx context.Context) ([]string, error) {
	c, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	imgs, err := c.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not list components: %v", err)
	}

	res := make([]string, len(imgs))
	for i, img := range imgs {
		res[i] = img.RepoTags[0]
	}
	return res, nil
}

// Install installs a new component.
func Install(ctx context.Context, id string) error {
	c, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	rc, err := c.ImagePull(ctx, id, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("could not pull %s: %v", id, err)
	}
	defer rc.Close()
	io.Copy(ioutil.Discard, rc)

	return nil
}
