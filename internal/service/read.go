package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Denxuan/sdk/internal/catalog"
	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
)

type ReadService struct {
	Store   *store.Store
	Catalog *catalog.Client
}

func NewRead(home string) *ReadService {
	return &ReadService{Store: store.New(home), Catalog: catalog.New()}
}

type Installed struct {
	Tool        model.Tool `json:"tool"`
	Version     string     `json:"version"`
	Path        string     `json:"path"`
	Managed     bool       `json:"managed"`
	Current     bool       `json:"current"`
	InstalledAt string     `json:"installedAt,omitempty"`
}

type Current struct {
	Tool    model.Tool `json:"tool"`
	Version string     `json:"version"`
	Path    string     `json:"path"`
	Link    string     `json:"link"`
}

type Available struct {
	Tool      model.Tool `json:"tool"`
	Version   string     `json:"version"`
	LTS       bool       `json:"lts"`
	Installed bool       `json:"installed"`
	Current   bool       `json:"current"`
}

type Project struct {
	Found    bool                  `json:"found"`
	File     string                `json:"file,omitempty"`
	Versions map[model.Tool]string `json:"versions"`
}

type Doctor struct {
	Passed   int     `json:"passed"`
	Warnings int     `json:"warnings"`
	Errors   int     `json:"errors"`
	Checks   []Check `json:"checks"`
}

type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type Update struct {
	Tool      model.Tool `json:"tool"`
	Current   string     `json:"current,omitempty"`
	Latest    string     `json:"latest"`
	Available bool       `json:"available"`
}

func (s *ReadService) ListInstalled(ctx context.Context, selected *model.Tool) ([]Installed, error) {
	state, err := s.Store.Load()
	if err != nil {
		return nil, err
	}
	tools := model.Tools()
	if selected != nil {
		tools = []model.Tool{*selected}
	}
	var result []Installed
	for _, tool := range tools {
		for _, item := range state.Installed[tool] {
			result = append(result, Installed{Tool: tool, Version: item.Version, Path: item.Path, Managed: item.Managed, Current: state.Defaults[tool] == item.Version, InstalledAt: item.InstalledAt.Format("2006-01-02T15:04:05Z07:00")})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return string(result[i].Tool)+result[i].Version < string(result[j].Tool)+result[j].Version
	})
	return result, nil
}

func (s *ReadService) CurrentVersions(ctx context.Context) ([]Current, error) {
	state, err := s.Store.Load()
	if err != nil {
		return nil, err
	}
	var result []Current
	for _, tool := range model.Tools() {
		version := state.Defaults[tool]
		if version == "" {
			continue
		}
		installation, found := find(state.Installed[tool], version)
		if !found {
			continue
		}
		result = append(result, Current{Tool: tool, Version: version, Path: installation.Path, Link: s.Store.CurrentPath(tool)})
	}
	return result, nil
}

func (s *ReadService) AvailableVersions(ctx context.Context, selected model.Tool) ([]Available, error) {
	state, err := s.Store.Load()
	if err != nil {
		return nil, err
	}
	versions, err := s.Catalog.Versions(ctx, selected)
	if err != nil {
		return nil, err
	}
	result := make([]Available, 0, len(versions))
	for _, version := range versions {
		result = append(result, Available{Tool: selected, Version: version.Number, LTS: version.LTS, Installed: has(state.Installed[selected], version.Number), Current: state.Defaults[selected] == version.Number})
	}
	return result, nil
}

func (s *ReadService) CheckUpdates(ctx context.Context) ([]Update, error) {
	state, err := s.Store.Load()
	if err != nil {
		return nil, err
	}
	var result []Update
	for _, tool := range model.Tools() {
		if len(state.Installed[tool]) == 0 {
			continue
		}
		versions, err := s.Catalog.Versions(ctx, tool)
		if err != nil {
			return nil, err
		}
		if len(versions) == 0 {
			continue
		}
		latest := versions[0].Number
		current := state.Defaults[tool]
		result = append(result, Update{Tool: tool, Current: current, Latest: latest, Available: current == "" || catalog.CompareVersions(latest, current) > 0})
	}
	return result, nil
}

func (s *ReadService) ToolPath(ctx context.Context, tool model.Tool) (string, error) {
	state, err := s.Store.Load()
	if err != nil {
		return "", err
	}
	if state.Defaults[tool] == "" {
		return "", fmt.Errorf("no default %s version is set", tool)
	}
	return s.Store.CurrentPath(tool), nil
}

func (s *ReadService) ToolExecutables(ctx context.Context, tool model.Tool) (map[string]string, error) {
	path, err := s.ToolPath(ctx, tool)
	if err != nil {
		return nil, err
	}
	names := map[model.Tool][]string{model.Java: {"java"}, model.Maven: {"mvn"}, model.MVND: {"mvnd"}, model.Gradle: {"gradle"}, model.Go: {"go"}, model.NodeJS: {"node", "npm", "npx"}}[tool]
	result := make(map[string]string, len(names))
	for _, name := range names {
		executable := filepath.Join(path, "bin", name)
		if _, err := os.Stat(executable); err == nil {
			result[name] = executable
		}
	}
	return result, nil
}

func (s *ReadService) ProjectVersions(ctx context.Context, directory string) (Project, error) {
	if directory == "" {
		var err error
		directory, err = os.Getwd()
		if err != nil {
			return Project{}, err
		}
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		return Project{}, err
	}
	for {
		path := filepath.Join(directory, ".sdk-version")
		data, readErr := os.ReadFile(path)
		if readErr == nil {
			versions, parseErr := parseProject(data)
			if parseErr != nil {
				return Project{}, fmt.Errorf("parse %s: %w", path, parseErr)
			}
			return Project{Found: true, File: path, Versions: versions}, nil
		}
		if !errors.Is(readErr, os.ErrNotExist) {
			return Project{}, readErr
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return Project{Found: false, Versions: map[model.Tool]string{}}, nil
		}
		directory = parent
	}
}

func (s *ReadService) StateJSON(ctx context.Context) ([]byte, error) {
	state, err := s.Store.Load()
	if err != nil {
		return nil, err
	}
	return json.Marshal(state)
}

func (s *ReadService) DoctorReport(ctx context.Context) (Doctor, error) {
	report := Doctor{}
	check := func(name, status, detail string) {
		report.Checks = append(report.Checks, Check{Name: name, Status: status, Detail: detail})
		if status == "ok" {
			report.Passed++
		}
		if status == "warn" {
			report.Warnings++
		}
		if status == "error" {
			report.Errors++
		}
	}
	if info, err := os.Stat(s.Store.Home); err == nil && info.IsDir() {
		check("sdk_home", "ok", s.Store.Home)
	} else {
		check("sdk_home", "error", "SDK home is unavailable")
	}
	state, err := s.Store.Load()
	if err != nil {
		check("state", "error", err.Error())
		return report, nil
	}
	for _, tool := range model.Tools() {
		version := state.Defaults[tool]
		if version == "" {
			continue
		}
		installation, found := find(state.Installed[tool], version)
		if !found {
			check(string(tool)+"_current", "error", "default version is not registered")
			continue
		}
		if _, err := os.Stat(installation.Path); err != nil {
			check(string(tool)+"_path", "error", err.Error())
		} else {
			check(string(tool)+"_path", "ok", installation.Path)
		}
	}
	return report, nil
}

func parseProject(data []byte) (map[model.Tool]string, error) {
	result := make(map[model.Tool]string)
	for lineNumber, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("line %d must be tool=version", lineNumber+1)
		}
		tool := model.Tool(strings.TrimSpace(parts[0]))
		if !tool.Valid() {
			return nil, fmt.Errorf("line %d has unsupported tool %q", lineNumber+1, tool)
		}
		result[tool] = strings.TrimSpace(parts[1])
	}
	return result, nil
}

func find(installed []model.InstalledVersion, version string) (model.InstalledVersion, bool) {
	for _, item := range installed {
		if item.Version == version {
			return item, true
		}
	}
	return model.InstalledVersion{}, false
}
func has(installed []model.InstalledVersion, version string) bool {
	_, ok := find(installed, version)
	return ok
}
