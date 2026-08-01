package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
)

const nodeReleasesURL = "https://nodejs.org/dist/index.json"

func (c *Client) nodeVersions(ctx context.Context) ([]Version, error) {
	var response []struct {
		Version string          `json:"version"`
		LTS     json.RawMessage `json:"lts"`
	}
	if err := c.getJSON(ctx, nodeReleasesURL, &response); err != nil {
		return nil, err
	}
	versions := make([]Version, 0, len(response))
	for _, release := range response {
		versions = append(versions, Version{Number: strings.TrimPrefix(release.Version, "v"), LTS: nodeLTS(release.LTS)})
	}
	return stableReleases(versions), nil
}

func nodeLTS(value json.RawMessage) bool {
	return len(value) > 0 && string(value) != "false" && string(value) != "null" && string(value) != `""`
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
