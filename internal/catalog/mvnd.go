package catalog

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/Denxuan/sdk/internal/installer"
)

const mvndReleasesURL = "https://dlcdn.apache.org/maven/mvnd/"

var mvndReleasePattern = regexp.MustCompile(`href="([0-9]+(?:\.[0-9]+)+(?:[-][^/"]+)*)/"`)

func (c *Client) mvndVersions(ctx context.Context) ([]Version, error) {
	index, err := c.getText(ctx, mvndReleasesURL)
	if err != nil {
		return nil, err
	}
	matches := mvndReleasePattern.FindAllStringSubmatch(index, -1)
	versions := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) == 2 {
			versions = append(versions, match[1])
		}
	}
	return stableVersions(versions), nil
}

func (c *Client) mvndArtifact(ctx context.Context, version string) (Artifact, error) {
	platform, err := mvndPlatform()
	if err != nil {
		return Artifact{}, err
	}
	name := fmt.Sprintf("maven-mvnd-%s-%s.tar.gz", version, platform)
	url := mvndReleasesURL + version + "/" + name
	checksumText, err := c.getText(ctx, url+".sha256")
	if err != nil {
		return Artifact{}, err
	}
	checksum := strings.Fields(checksumText)
	if len(checksum) == 0 {
		return Artifact{}, fmt.Errorf("maven-mvnd checksum is unavailable for %s", version)
	}
	return Artifact{URL: url, Checksum: installer.Checksum{Algorithm: "sha256", Value: checksum[0]}}, nil
}

func mvndPlatform() (string, error) {
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		return "windows-amd64", nil
	}
	if (runtime.GOOS == "darwin" || runtime.GOOS == "linux") && runtime.GOARCH == "amd64" {
		return runtime.GOOS + "-amd64", nil
	}
	if (runtime.GOOS == "darwin" || runtime.GOOS == "linux") && runtime.GOARCH == "arm64" {
		return runtime.GOOS + "-aarch64", nil
	}
	return "", fmt.Errorf("maven-mvnd has no official archive for %s/%s", runtime.GOOS, runtime.GOARCH)
}
