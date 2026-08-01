package catalog

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

const adoptiumReleasesURL = "https://api.adoptium.net/v3/info/available_releases"

func (c *Client) javaVersions(ctx context.Context) ([]string, error) {
	var response struct {
		AvailableReleases []int `json:"available_releases"`
	}
	if err := c.getJSON(ctx, adoptiumReleasesURL, &response); err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(response.AvailableReleases))
	for _, release := range response.AvailableReleases {
		versions = append(versions, fmt.Sprint(release))
	}
	return unique(versions), nil
}

func javaArtifact(version string) (Artifact, error) {
	osName, supported := map[string]string{"darwin": "mac", "linux": "linux", "windows": "windows"}[runtime.GOOS]
	if !supported {
		return Artifact{}, fmt.Errorf("java is not supported on %s", runtime.GOOS)
	}
	architecture := runtime.GOARCH
	if architecture == "arm64" {
		architecture = "aarch64"
	}
	featureVersion := strings.SplitN(version, ".", 2)[0]
	url := fmt.Sprintf("https://api.adoptium.net/v3/binary/latest/%s/ga/%s/%s/jdk/hotspot/normal/eclipse", featureVersion, osName, architecture)
	return Artifact{URL: url}, nil
}
