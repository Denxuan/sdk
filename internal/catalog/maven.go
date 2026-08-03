package catalog

import (
	"context"
	"fmt"
	"strings"

	"github.com/Denxuan/sdk/internal/installer"
)

const mavenMetadataURL = "https://repo.maven.apache.org/maven2/org/apache/maven/apache-maven/maven-metadata.xml"

func (c *Client) mavenVersions(ctx context.Context) ([]Version, error) {
	var response struct {
		Versioning struct {
			Versions []string `xml:"versions>version"`
		} `xml:"versioning"`
	}
	if err := c.getXML(ctx, mavenMetadataURL, &response); err != nil {
		return nil, err
	}
	return stableVersions(response.Versioning.Versions), nil
}

func (c *Client) mavenArtifact(ctx context.Context, version string) (Artifact, error) {
	url := fmt.Sprintf("https://archive.apache.org/dist/maven/maven-3/%s/binaries/apache-maven-%s-bin.tar.gz", version, version)
	checksumFile, err := c.getText(ctx, url+".sha512")
	if err != nil {
		return Artifact{}, err
	}
	checksum := strings.Fields(checksumFile)
	if len(checksum) == 0 {
		return Artifact{}, fmt.Errorf("Maven checksum is unavailable for %s", version)
	}
	return Artifact{URL: url, Checksum: installer.Checksum{Algorithm: "sha512", Value: checksum[0]}}, nil
}
