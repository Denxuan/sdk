package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Denxuan/sdk/internal/model"
)

type Store struct{ Home string }

func New(home string) *Store { return &Store{Home: home} }

func (s *Store) ToolsDir() string  { return filepath.Join(s.Home, "tools") }
func (s *Store) StatePath() string { return filepath.Join(s.Home, "state.json") }

func emptyState() model.State {
	return model.State{Defaults: make(map[model.Tool]string), Installed: make(map[model.Tool][]model.InstalledVersion)}
}

func (s *Store) Load() (model.State, error) {
	data, err := os.ReadFile(s.StatePath())
	if errors.Is(err, os.ErrNotExist) {
		return emptyState(), nil
	}
	if err != nil {
		return model.State{}, fmt.Errorf("read state: %w", err)
	}
	var state model.State
	if err := json.Unmarshal(data, &state); err != nil {
		return model.State{}, fmt.Errorf("parse state: %w", err)
	}
	if state.Defaults == nil {
		state.Defaults = make(map[model.Tool]string)
	}
	if state.Installed == nil {
		state.Installed = make(map[model.Tool][]model.InstalledVersion)
	}
	return state, nil
}

func (s *Store) Save(state model.State) error {
	if err := os.MkdirAll(s.Home, 0755); err != nil {
		return fmt.Errorf("create sdk home: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	tmp, err := os.CreateTemp(s.Home, "state-*.json")
	if err != nil {
		return fmt.Errorf("create temporary state: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Chmod(0644)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	if err := os.Rename(tmpName, s.StatePath()); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}
