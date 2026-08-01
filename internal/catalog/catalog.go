package catalog

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Denxuan/sdk/internal/model"
)

type Client struct {
	HTTP *http.Client
}

type Artifact struct {
	URL string
}

type Version struct {
	Number string
	LTS    bool
}

func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) Versions(ctx context.Context, tool model.Tool) ([]Version, error) {
	switch tool {
	case model.Java:
		return c.javaVersions(ctx)
	case model.Maven:
		return c.mavenVersions(ctx)
	case model.Go:
		return c.goVersions(ctx)
	case model.NodeJS:
		return c.nodeVersions(ctx)
	default:
		return nil, fmt.Errorf("remote catalogue for %s is not connected", tool)
	}
}

func (c *Client) Artifact(tool model.Tool, version string) (Artifact, error) {
	switch tool {
	case model.Java:
		return javaArtifact(version)
	case model.Maven:
		return mavenArtifact(version)
	case model.Go:
		return goArtifact(version)
	case model.NodeJS:
		return nodeArtifact(version)
	default:
		return Artifact{}, fmt.Errorf("unsupported tool %q", tool)
	}
}
