package catalog

import (
	"context"
	"fmt"

	"github.com/Denxuan/sdk/internal/installer"
)

const gradleVersionsURL = "https://services.gradle.org/versions/all"

type gradleRelease struct {
	Version     string `json:"version"`
	Final       bool   `json:"final"`
	Snapshot    bool   `json:"snapshot"`
	Nightly     bool   `json:"nightly"`
	DownloadURL string `json:"downloadUrl"`
	Checksum    string `json:"checksum"`
}

func (c *Client) gradleReleases(ctx context.Context) ([]gradleRelease, error) {
	var releases []gradleRelease
	if err := c.getJSON(ctx, gradleVersionsURL, &releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func (c *Client) gradleVersions(ctx context.Context) ([]Version, error) {
	releases, err := c.gradleReleases(ctx)
	if err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(releases))
	for _, release := range releases {
		if release.Final && !release.Snapshot && !release.Nightly {
			versions = append(versions, release.Version)
		}
	}
	return stableVersions(versions), nil
}

func (c *Client) gradleArtifact(ctx context.Context, version string) (Artifact, error) {
	releases, err := c.gradleReleases(ctx)
	if err != nil {
		return Artifact{}, err
	}
	for _, release := range releases {
		if release.Version == version && release.Final && release.Checksum != "" && release.DownloadURL != "" {
			return Artifact{URL: release.DownloadURL, Checksum: installer.Checksum{Algorithm: "sha256", Value: release.Checksum}}, nil
		}
	}
	return Artifact{}, fmt.Errorf("no stable Gradle archive with checksum for %s", version)
}
