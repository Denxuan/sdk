package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/Denxuan/sdk/internal/installer"
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

func (c *Client) nodeArtifact(ctx context.Context, version string) (Artifact, error) {
	osName := runtime.GOOS
	if osName == "windows" {
		osName = "win"
	}
	extension := "tar.gz"
	if osName == "win" {
		extension = "zip"
	}
	filename := fmt.Sprintf("node-v%s-%s-%s.%s", version, osName, runtime.GOARCH, extension)
	baseURL := fmt.Sprintf("https://nodejs.org/dist/v%s", version)
	manifest, err := c.getText(ctx, baseURL+"/SHASUMS256.txt")
	if err != nil {
		return Artifact{}, err
	}
	checksum, found := manifestChecksum(manifest, filename)
	if !found {
		return Artifact{}, fmt.Errorf("Node.js checksum is unavailable for %s", filename)
	}
	return Artifact{URL: baseURL + "/" + filename, Checksum: installer.Checksum{Algorithm: "sha256", Value: checksum}}, nil
}

func manifestChecksum(manifest, filename string) (string, bool) {
	for _, line := range strings.Split(manifest, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == filename {
			return fields[0], true
		}
	}
	return "", false
}
