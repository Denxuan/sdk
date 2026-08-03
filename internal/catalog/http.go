package catalog

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

const maxCatalogueResponseSize = 10 << 20

type StatusError struct {
	URL        string
	StatusCode int
	Status     string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("request %s: server returned %s", e.URL, e.Status)
}

func (c *Client) getJSON(ctx context.Context, url string, destination any) error {
	return c.decode(ctx, url, func(reader io.Reader) error {
		return json.NewDecoder(reader).Decode(destination)
	})
}
func (c *Client) getXML(ctx context.Context, url string, destination any) error {
	return c.decode(ctx, url, func(reader io.Reader) error {
		return xml.NewDecoder(reader).Decode(destination)
	})
}

func (c *Client) getText(ctx context.Context, url string) (string, error) {
	var text string
	err := c.decode(ctx, url, func(reader io.Reader) error {
		data, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		text = string(data)
		return nil
	})
	return text, err
}

func (c *Client) decode(ctx context.Context, url string, decode func(io.Reader) error) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return fmt.Errorf("request %s: %w", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return &StatusError{URL: url, StatusCode: response.StatusCode, Status: response.Status}
	}
	if err := decode(io.LimitReader(response.Body, maxCatalogueResponseSize)); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}
	return nil
}
