package catalog

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"strconv"
	"strings"

	"github.com/Denxuan/sdk/internal/installer"
)

const adoptiumReleasesURL = "https://api.adoptium.net/v3/info/available_releases"

func (c *Client) javaVersions(ctx context.Context) ([]Version, error) {
	var response struct {
		AvailableReleases []int `json:"available_releases"`
		AvailableLTS      []int `json:"available_lts_releases"`
	}
	if err := c.getJSON(ctx, adoptiumReleasesURL, &response); err != nil {
		return nil, err
	}
	ltsFeatures := make(map[int]bool, len(response.AvailableLTS))
	for _, featureVersion := range response.AvailableLTS {
		ltsFeatures[featureVersion] = true
	}
	versions := make([]Version, 0, len(response.AvailableReleases))
	for _, release := range response.AvailableReleases {
		version, err := c.javaLatestVersion(ctx, release)
		if err != nil {
			return nil, err
		}
		if version == "" {
			continue
		}
		versions = append(versions, Version{Number: version, LTS: ltsFeatures[release]})
	}
	return stableReleases(versions), nil
}

func (c *Client) javaLatestVersion(ctx context.Context, featureVersion int) (string, error) {
	osName, architecture, err := javaPlatform()
	if err != nil {
		return "", err
	}
	var response []struct {
		VersionData struct {
			Semver string `json:"semver"`
		} `json:"version_data"`
	}
	if err := c.getJSON(ctx, javaAssetsURL(featureVersion, osName, architecture), &response); err != nil {
		var statusError *StatusError
		if errors.As(err, &statusError) && statusError.StatusCode == 404 {
			return "", nil
		}
		return "", err
	}
	if len(response) == 0 || response[0].VersionData.Semver == "" {
		return "", fmt.Errorf("no Java %d release is available for %s/%s", featureVersion, osName, architecture)
	}
	return strings.SplitN(response[0].VersionData.Semver, "+", 2)[0], nil
}

func javaAssetsURL(featureVersion int, osName, architecture string) string {
	return javaAssetsURLWithPageSize(featureVersion, osName, architecture, 1)
}

func javaAssetsURLWithPageSize(featureVersion int, osName, architecture string, pageSize int) string {
	query := url.Values{
		"architecture": {architecture},
		"image_type":   {"jdk"},
		"jvm_impl":     {"hotspot"},
		"os":           {osName},
		"page_size":    {fmt.Sprintf("%d", pageSize)},
		"sort_order":   {"DESC"},
		"vendor":       {"eclipse"},
	}
	return fmt.Sprintf("https://api.adoptium.net/v3/assets/feature_releases/%d/ga?%s", featureVersion, query.Encode())
}

func (c *Client) javaArtifact(ctx context.Context, version string) (Artifact, error) {
	osName, architecture, err := javaPlatform()
	if err != nil {
		return Artifact{}, err
	}
	featureVersion := strings.SplitN(version, ".", 2)[0]
	feature, err := strconv.Atoi(featureVersion)
	if err != nil {
		return Artifact{}, fmt.Errorf("parse Java feature version %q: %w", featureVersion, err)
	}
	var response []struct {
		Binary struct {
			Package struct {
				Link     string `json:"link"`
				Checksum string `json:"checksum"`
			} `json:"package"`
		} `json:"binary"`
		VersionData struct {
			Semver string `json:"semver"`
		} `json:"version_data"`
	}
	if err := c.getJSON(ctx, javaAssetsURLWithPageSize(feature, osName, architecture, 100), &response); err != nil {
		return Artifact{}, err
	}
	for _, release := range response {
		releaseVersion := strings.SplitN(release.VersionData.Semver, "+", 2)[0]
		if releaseVersion == version && release.Binary.Package.Link != "" && release.Binary.Package.Checksum != "" {
			return Artifact{URL: release.Binary.Package.Link, Checksum: installer.Checksum{Algorithm: "sha256", Value: release.Binary.Package.Checksum}}, nil
		}
	}
	return Artifact{}, fmt.Errorf("no Java archive with checksum for %s/%s %s", osName, architecture, version)
}

func javaPlatform() (string, string, error) {
	osName, supported := map[string]string{"darwin": "mac", "linux": "linux", "windows": "windows"}[runtime.GOOS]
	if !supported {
		return "", "", fmt.Errorf("java is not supported on %s", runtime.GOOS)
	}
	architecture := runtime.GOARCH
	if architecture == "arm64" {
		architecture = "aarch64"
	}
	return osName, architecture, nil
}
