package catalog

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

const goReleasesURL = "https://go.dev/dl/?mode=json&include=all"

func (c *Client) goVersions(ctx context.Context) ([]Version, error) {
	var response []struct {
		Version string `json:"version"`
	}
	if err := c.getJSON(ctx, goReleasesURL, &response); err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(response))
	for _, release := range response {
		versions = append(versions, strings.TrimPrefix(release.Version, "go"))
	}
	return stableVersions(versions), nil
}

func goArtifact(version string) (Artifact, error) {
	extension := "tar.gz"
	if runtime.GOOS == "windows" {
		extension = "zip"
	}
	url := fmt.Sprintf("https://go.dev/dl/go%s.%s-%s.%s", version, runtime.GOOS, runtime.GOARCH, extension)
	return Artifact{URL: url}, nil
}
