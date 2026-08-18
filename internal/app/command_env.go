package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
)

func printEnvironment(stateStore *store.Store, args []string, out io.Writer) error {
	useProject, err := parseProjectFlag(args)
	if err != nil {
		return err
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	targets, err := environmentTargets(stateStore, state, useProject)
	if err != nil {
		return err
	}
	pathEntries := make([]string, 0, len(model.Tools()))
	for _, tool := range model.Tools() {
		target, found := targets[tool]
		if !found {
			continue
		}
		for _, variable := range environmentVariables(tool, target.Path) {
			fmt.Fprintf(out, "export %s=%q\n", variable.name, variable.value)
		}
		pathEntries = append(pathEntries, filepath.Join(target.Path, "bin"))
		if tool == model.Go {
			pathEntries = append(pathEntries, goToolPaths()...)
		}
	}
	if len(pathEntries) > 0 {
		fmt.Fprintf(out, "export PATH=%q:$PATH\n", strings.Join(pathEntries, ":"))
	}
	return nil
}

func goToolPaths() []string {
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		return []string{gobin}
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		gopath = filepath.Join(home, "go")
	}
	paths := filepath.SplitList(gopath)
	for index := range paths {
		paths[index] = filepath.Join(paths[index], "bin")
	}
	return paths
}

type environmentTarget struct {
	Version string
	Path    string
}

func environmentTargets(stateStore *store.Store, state model.State, useProject bool) (map[model.Tool]environmentTarget, error) {
	targets := make(map[model.Tool]environmentTarget, len(model.Tools()))
	for _, tool := range model.Tools() {
		version := state.Defaults[tool]
		if version == "" {
			continue
		}
		targets[tool] = environmentTarget{Version: version, Path: stateStore.CurrentPath(tool)}
	}
	if !useProject {
		return targets, nil
	}
	file, versions, found, err := findProjectVersions()
	if err != nil {
		return nil, err
	}
	if !found {
		return targets, nil
	}
	for tool, version := range versions {
		installation, installed := findInstallation(state.Installed[tool], version)
		if !installed {
			return nil, fmt.Errorf("project file %s requires %s %s, but it is not installed", file, tool, version)
		}
		targets[tool] = environmentTarget{Version: version, Path: installation.Path}
	}
	return targets, nil
}

func parseProjectFlag(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if len(args) == 1 && args[0] == "--project" {
		return true, nil
	}
	return false, errors.New("usage: sdk env [--project]")
}

func currentWorkingDirectory() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("find current directory: %w", err)
	}
	return directory, nil
}

type environmentVariable struct {
	name  string
	value string
}

func environmentVariables(tool model.Tool, currentPath string) []environmentVariable {
	switch tool {
	case model.Java:
		return []environmentVariable{{name: "JAVA_HOME", value: currentPath}}
	case model.Maven:
		return []environmentVariable{{name: "MAVEN_HOME", value: currentPath}, {name: "M2_HOME", value: currentPath}}
	case model.MVND:
		return []environmentVariable{{name: "MVND_HOME", value: currentPath}}
	case model.Go:
		return []environmentVariable{{name: "GOROOT", value: currentPath}}
	case model.NodeJS:
		return []environmentVariable{{name: "NODE_HOME", value: currentPath}}
	default:
		return nil
	}
}
