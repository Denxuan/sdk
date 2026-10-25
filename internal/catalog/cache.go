package catalog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/Denxuan/sdk/internal/model"
)

type versionCache struct {
	FetchedAt time.Time `json:"fetchedAt"`
	Versions  []Version `json:"versions"`
}

func (c *Client) readVersionsCache(tool model.Tool) ([]Version, time.Time, error) {
	data, err := os.ReadFile(filepath.Join(c.CacheDir, string(tool)+".json"))
	if err != nil {
		return nil, time.Time{}, err
	}
	var cache versionCache
	if err := json.Unmarshal(data, &cache); err != nil || cache.FetchedAt.IsZero() || cache.Versions == nil {
		if err == nil {
			err = os.ErrInvalid
		}
		return nil, time.Time{}, err
	}
	return cache.Versions, cache.FetchedAt, nil
}

func (c *Client) writeVersionsCache(tool model.Tool, versions []Version) error {
	if err := os.MkdirAll(c.CacheDir, 0755); err != nil {
		return err
	}
	file, err := os.CreateTemp(c.CacheDir, ".versions-*.tmp")
	if err != nil {
		return err
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {

		}
	}(file.Name())
	if err := json.NewEncoder(file).Encode(versionCache{FetchedAt: time.Now(), Versions: versions}); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	destination := filepath.Join(c.CacheDir, string(tool)+".json")
	if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(file.Name(), destination)
}
