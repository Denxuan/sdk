package catalog

import (
	"context"
	"fmt"
)

const mavenMetadataURL = "https://repo.maven.apache.org/maven2/org/apache/maven/apache-maven/maven-metadata.xml"

func (c *Client) mavenVersions(ctx context.Context) ([]string, error) {
	var response struct {
		Versioning struct {
			Versions []string `xml:"versions>version"`
		} `xml:"versioning"`
	}
	if err := c.getXML(ctx, mavenMetadataURL, &response); err != nil {
		return nil, err
	}
	return unique(response.Versioning.Versions), nil
}

func mavenArtifact(version string) (Artifact, error) {
	return Artifact{URL: fmt.Sprintf("https://archive.apache.org/dist/maven/maven-3/%s/binaries/apache-maven-%s-bin.tar.gz", version, version)}, nil
}
