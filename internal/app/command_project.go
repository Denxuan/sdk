package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
)

const projectVersionFile = ".sdk-version"

func project(ctx context.Context, stateStore *store.Store, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: sdk project <init|set|unset|remove|list>")
	}
	switch args[0] {
	case "init":
		if len(args) != 1 {
			return errors.New("usage: sdk project init")
		}
		return initializeProject(stateStore, out)
	case "set":
		if len(args) != 3 {
			return errors.New("usage: sdk project set <tool> <version>")
		}
		return setProjectVersion(stateStore, args[1], args[2], out)
	case "install":
		if len(args) != 1 {
			return errors.New("usage: sdk project install")
		}
		return installProject(ctx, stateStore, out)
	case "unset":
		if len(args) != 2 {
			return errors.New("usage: sdk project unset <tool>")
		}
		return unsetProjectVersion(args[1], out)
	case "remove":
		if len(args) != 1 {
			return errors.New("usage: sdk project remove")
		}
		return removeProjectFile(out)
	case "list":
		if len(args) != 1 {
			return errors.New("usage: sdk project list")
		}
		return listProjectVersions(out)
	default:
		return fmt.Errorf("unknown project command %q (use init, set, unset, remove, or list)", args[0])
	}
}

func removeProjectFile(out io.Writer) error {
	path, found, err := findProjectFile()
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no %s found in this directory or its parents", projectVersionFile)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove project file %s: %w", path, err)
	}
	_, _ = fmt.Fprintf(out, "removed %s\n", path)
	return nil
}

func unsetProjectVersion(toolName string, out io.Writer) error {
	tool, err := parseTool(toolName)
	if err != nil {
		return err
	}
	directory, err := currentWorkingDirectory()
	if err != nil {
		return err
	}
	path := filepath.Join(directory, projectVersionFile)
	versions, err := readProjectVersions(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("no %s in current directory", projectVersionFile)
		}
		return err
	}
	if _, found := versions[tool]; !found {
		return fmt.Errorf("%s is not set in %s", tool, path)
	}
	delete(versions, tool)
	if err := writeProjectVersions(path, versions); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "unset %s in %s\n", tool, path)
	return nil
}

func installProject(ctx context.Context, stateStore *store.Store, out io.Writer) error {
	path, versions, found, err := findProjectVersions()
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no %s found in this directory or its parents", projectVersionFile)
	}
	if len(versions) == 0 {
		return fmt.Errorf("no tool versions are set in %s", path)
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	for _, tool := range model.Tools() {
		version, found := versions[tool]
		if !found {
			continue
		}
		version = normalizeToolVersion(tool, version)
		if hasVersion(state.Installed[tool], version) {
			_, _ = fmt.Fprintf(out, "%s %s is already installed\n", tool, version)
			continue
		}
		if err := install(ctx, stateStore, []string{string(tool), version}, out, false); err != nil {
			return fmt.Errorf("install project %s %s: %w", tool, version, err)
		}
	}
	return nil
}

func initializeProject(stateStore *store.Store, out io.Writer) error {
	directory, err := currentWorkingDirectory()
	if err != nil {
		return err
	}
	path := filepath.Join(directory, projectVersionFile)
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("project file already exists: %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect project file: %w", err)
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	versions := make(map[model.Tool]string)
	for _, tool := range model.Tools() {
		if version := state.Defaults[tool]; version != "" {
			versions[tool] = version
		}
	}
	if len(versions) == 0 {
		return errors.New("no default versions are set; use sdk default before sdk project init")
	}
	if err := writeProjectVersions(path, versions); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "created %s\n", path)
	return nil
}

func setProjectVersion(stateStore *store.Store, toolName, version string, out io.Writer) error {
	tool, err := parseTool(toolName)
	if err != nil {
		return err
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	if !hasVersion(state.Installed[tool], version) {
		version = normalizeToolVersion(tool, version)
	}
	if !hasVersion(state.Installed[tool], version) {
		return fmt.Errorf("%s %s is not installed", tool, version)
	}
	directory, err := currentWorkingDirectory()
	if err != nil {
		return err
	}
	path := filepath.Join(directory, projectVersionFile)
	versions, err := readProjectVersions(path)
	if errors.Is(err, os.ErrNotExist) {
		versions = make(map[model.Tool]string)
	} else if err != nil {
		return err
	}
	versions[tool] = version
	if err := writeProjectVersions(path, versions); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "set %s %s in %s\n", tool, version, path)
	return nil
}

func listProjectVersions(out io.Writer) error {
	path, versions, found, err := findProjectVersions()
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no %s found in this directory or its parents", projectVersionFile)
	}
	_, _ = fmt.Fprintln(out, path)
	for _, tool := range model.Tools() {
		if version := versions[tool]; version != "" {
			_, _ = fmt.Fprintf(out, "%s=%s\n", tool, version)
		}
	}
	return nil
}

func findProjectVersions() (string, map[model.Tool]string, bool, error) {
	path, found, err := findProjectFile()
	if err != nil || !found {
		return "", nil, false, err
	}
	versions, err := readProjectVersions(path)
	if err != nil {
		return "", nil, false, err
	}
	return path, versions, true, nil
}

func findProjectFile() (string, bool, error) {
	directory, err := currentWorkingDirectory()
	if err != nil {
		return "", false, err
	}
	for {
		path := filepath.Join(directory, projectVersionFile)
		if _, err := os.Stat(path); err == nil {
			return path, true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", false, err
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", false, nil
		}
		directory = parent
	}
}

func readProjectVersions(path string) (map[model.Tool]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	versions := make(map[model.Tool]string)
	for lineNumber, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("parse %s line %d: expected tool=version", path, lineNumber+1)
		}
		tool, err := parseTool(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("parse %s line %d: %w", path, lineNumber+1, err)
		}
		if _, duplicate := versions[tool]; duplicate {
			return nil, fmt.Errorf("parse %s line %d: duplicate %s", path, lineNumber+1, tool)
		}
		versions[tool] = strings.TrimSpace(parts[1])
	}
	return versions, nil
}

func writeProjectVersions(path string, versions map[model.Tool]string) error {
	tools := make([]string, 0, len(versions))
	for tool := range versions {
		tools = append(tools, string(tool))
	}
	sort.Strings(tools)
	lines := make([]string, 0, len(tools))
	for _, name := range tools {
		lines = append(lines, fmt.Sprintf("%s=%s", name, versions[model.Tool(name)]))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return fmt.Errorf("write project file: %w", err)
	}
	return nil
}
