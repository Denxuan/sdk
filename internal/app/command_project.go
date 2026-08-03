package app

import (
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

func project(stateStore *store.Store, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: sdk project <init|set|list>")
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
	case "list":
		if len(args) != 1 {
			return errors.New("usage: sdk project list")
		}
		return listProjectVersions(out)
	default:
		return fmt.Errorf("unknown project command %q (use init, set, or list)", args[0])
	}
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
	fmt.Fprintf(out, "created %s\n", path)
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
	fmt.Fprintf(out, "set %s %s in %s\n", tool, version, path)
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
	fmt.Fprintln(out, path)
	for _, tool := range model.Tools() {
		if version := versions[tool]; version != "" {
			fmt.Fprintf(out, "%s=%s\n", tool, version)
		}
	}
	return nil
}

func findProjectVersions() (string, map[model.Tool]string, bool, error) {
	directory, err := currentWorkingDirectory()
	if err != nil {
		return "", nil, false, err
	}
	for {
		path := filepath.Join(directory, projectVersionFile)
		versions, err := readProjectVersions(path)
		if err == nil {
			return path, versions, true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", nil, false, err
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", nil, false, nil
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
