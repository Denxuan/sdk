package catalog

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Denxuan/sdk/internal/installer"
	"github.com/Denxuan/sdk/internal/model"
)

type Client struct {
	HTTP *http.Client
}

type Artifact struct {
	URL      string
	Checksum installer.Checksum
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

func (c *Client) Artifact(ctx context.Context, tool model.Tool, version string) (Artifact, error) {
	switch tool {
	case model.Java:
		return c.javaArtifact(ctx, version)
	case model.Maven:
		return c.mavenArtifact(ctx, version)
	case model.Go:
		return c.goArtifact(ctx, version)
	case model.NodeJS:
		return c.nodeArtifact(ctx, version)
	default:
		return Artifact{}, fmt.Errorf("unsupported tool %q", tool)
	}
}
