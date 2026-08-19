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
const foojayZuluURL = "https://api.foojay.io/disco/v3.0/packages"

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
		versions = append(versions, Version{Number: version + "-tem", LTS: ltsFeatures[release]})
	}
	zulu, err := c.javaZuluVersions(ctx)
	if err != nil {
		return nil, err
	}
	return stableReleases(append(versions, zulu...)), nil
}

type zuluPackage struct {
	ID             string `json:"id"`
	JavaVersion    string `json:"java_version"`
	TermOfSupport  string `json:"term_of_support"`
	DirectDownload string `json:"direct_download_uri"`
	PackageInfoURI string `json:"pkg_info_uri"`
	Links          struct {
		PackageInfo string `json:"pkg_info_uri"`
	} `json:"links"`
}

func (c *Client) javaZuluPackages(ctx context.Context) ([]zuluPackage, error) {
	osName, architecture, err := foojayPlatform()
	if err != nil {
		return nil, err
	}
	query := url.Values{
		"distribution":     {"zulu"},
		"architecture":     {architecture},
		"operating_system": {osName},
		"package_type":     {"jdk"},
		"javafx_bundled":   {"true"},
		"release_status":   {"ga"},
		"archive_type":     {"zip"},
		"latest":           {"available"},
		"limit":            {"100"},
	}
	var response struct {
		Result []zuluPackage `json:"result"`
	}
	if err := c.getJSON(ctx, foojayZuluURL+"?"+query.Encode(), &response); err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (c *Client) javaZuluVersions(ctx context.Context) ([]Version, error) {
	packages, err := c.javaZuluPackages(ctx)
	if err != nil {
		return nil, err
	}
	versions := make([]Version, 0, len(packages))
	for _, item := range packages {
		version := strings.SplitN(item.JavaVersion, "+", 2)[0]
		if version == "" {
			continue
		}
		versions = append(versions, Version{Number: version + "-zulu", LTS: item.TermOfSupport == "lts"})
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
	if strings.HasSuffix(version, "-zulu") {
		version = strings.TrimSuffix(version, "-zulu")
		return c.javaZuluArtifact(ctx, version)
	}
	version = strings.TrimSuffix(version, "-tem")
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

func (c *Client) javaZuluArtifact(ctx context.Context, version string) (Artifact, error) {
	packages, err := c.javaZuluPackages(ctx)
	if err != nil {
		return Artifact{}, err
	}
	for _, item := range packages {
		if strings.SplitN(item.JavaVersion, "+", 2)[0] != version {
			continue
		}
		infoURL := item.PackageInfoURI
		if infoURL == "" {
			infoURL = item.Links.PackageInfo
		}
		if infoURL == "" {
			continue
		}
		var details struct {
			Result []struct {
				DirectDownload string `json:"direct_download_uri"`
				Checksum       string `json:"checksum"`
				ChecksumType   string `json:"checksum_type"`
			} `json:"result"`
		}
		if err := c.getJSON(ctx, infoURL, &details); err != nil {
			return Artifact{}, err
		}
		if len(details.Result) > 0 && details.Result[0].DirectDownload != "" && details.Result[0].Checksum != "" {
			algorithm := details.Result[0].ChecksumType
			if algorithm == "" {
				algorithm = "sha256"
			}
			return Artifact{URL: details.Result[0].DirectDownload, Checksum: installer.Checksum{Algorithm: algorithm, Value: details.Result[0].Checksum}}, nil
		}
	}
	return Artifact{}, fmt.Errorf("no Azul Zulu JavaFX archive with checksum for %s", version)
}

func foojayPlatform() (string, string, error) {
	osName, supported := map[string]string{"darwin": "macos", "linux": "linux", "windows": "windows"}[runtime.GOOS]
	if !supported {
		return "", "", fmt.Errorf("Zulu Java is not supported on %s", runtime.GOOS)
	}
	architecture := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[runtime.GOARCH]
	if architecture == "" {
		return "", "", fmt.Errorf("Zulu Java is not supported on %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return osName, architecture, nil
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
