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
func (s *Store) CurrentPath(tool model.Tool) string {
	return filepath.Join(s.ToolsDir(), string(tool), "current")
}

// SetCurrent updates the managed current symlink without ever removing a real
// directory that was not created as a symlink by sdk.
func (s *Store) SetCurrent(tool model.Tool, target string) error {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve current target: %w", err)
	}
	current := s.CurrentPath(tool)
	if err := os.MkdirAll(filepath.Dir(current), 0755); err != nil {
		return fmt.Errorf("create current directory: %w", err)
	}
	if info, err := os.Lstat(current); err == nil && info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("refusing to replace non-symlink path %s", current)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect current link: %w", err)
	}
	temporary := current + ".next"
	if err := os.Remove(temporary); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove temporary current link: %w", err)
	}
	if err := os.Symlink(absTarget, temporary); err != nil {
		return fmt.Errorf("create current link: %w", err)
	}
	if err := os.Rename(temporary, current); err != nil {
		return fmt.Errorf("replace current link: %w", err)
	}
	return nil
}

func (s *Store) RemoveCurrent(tool model.Tool) error {
	current := s.CurrentPath(tool)
	info, err := os.Lstat(current)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect current link: %w", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("refusing to remove non-symlink path %s", current)
	}
	if err := os.Remove(current); err != nil {
		return fmt.Errorf("remove current link: %w", err)
	}
	return nil
}

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
