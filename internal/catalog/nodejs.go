package catalog

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

const nodeReleasesURL = "https://nodejs.org/dist/index.json"

func (c *Client) nodeVersions(ctx context.Context) ([]string, error) {
	var response []struct {
		Version string `json:"version"`
	}
	if err := c.getJSON(ctx, nodeReleasesURL, &response); err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(response))
	for _, release := range response {
		versions = append(versions, strings.TrimPrefix(release.Version, "v"))
	}
	return unique(versions), nil
}

func nodeArtifact(version string) (Artifact, error) {
	osName := runtime.GOOS
	if osName == "windows" {
		osName = "win"
	}
	extension := "tar.gz"
	if osName == "win" {
		extension = "zip"
	}
	url := fmt.Sprintf("https://nodejs.org/dist/v%s/node-v%s-%s-%s.%s", version, version, osName, runtime.GOARCH, extension)
	return Artifact{URL: url}, nil
}
