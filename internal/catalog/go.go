package catalog

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/Denxuan/sdk/internal/installer"
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

func (c *Client) goArtifact(ctx context.Context, version string) (Artifact, error) {
	var response []struct {
		Version string `json:"version"`
		Files   []struct {
			Filename string `json:"filename"`
			OS       string `json:"os"`
			Arch     string `json:"arch"`
			Kind     string `json:"kind"`
			SHA256   string `json:"sha256"`
		} `json:"files"`
	}
	if err := c.getJSON(ctx, goReleasesURL, &response); err != nil {
		return Artifact{}, err
	}
	for _, release := range response {
		if strings.TrimPrefix(release.Version, "go") != version {
			continue
		}
		for _, file := range release.Files {
			if file.OS == runtime.GOOS && file.Arch == runtime.GOARCH && file.Kind == "archive" && file.SHA256 != "" {
				return Artifact{URL: "https://go.dev/dl/" + file.Filename, Checksum: installer.Checksum{Algorithm: "sha256", Value: file.SHA256}}, nil
			}
		}
	}
	return Artifact{}, fmt.Errorf("no Go archive with checksum for %s/%s %s", runtime.GOOS, runtime.GOARCH, version)
}
