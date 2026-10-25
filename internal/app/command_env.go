package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/rustup"
	"github.com/Denxuan/sdk/internal/store"
)

func printEnvironment(stateStore *store.Store, args []string, out io.Writer) error {
	useProject, powershell, err := parseEnvironmentFlags(args)
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		powershell = true
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
	activeTools := make(map[model.Tool]bool, len(targets))
	for _, tool := range model.Tools() {
		target, found := targets[tool]
		if !found {
			continue
		}
		activeTools[tool] = true
		for _, variable := range environmentVariables(tool, target.Path) {
			writeEnvironmentVariable(out, variable, powershell)
		}
		if tool == model.Rust {
			pathEntries = append(pathEntries, rustup.CargoBin())
			continue
		}
		pathEntries = append(pathEntries, installationBinPath(tool, target.Path))
		if tool == model.Go {
			pathEntries = append(pathEntries, goToolPaths()...)
		}
	}
	for _, tool := range model.Tools() {
		if activeTools[tool] {
			continue
		}
		for _, variable := range environmentVariables(tool, "") {
			if powershell {
				_, _ = fmt.Fprintf(out, "Remove-Item Env:%s -ErrorAction SilentlyContinue\n", variable.name)
			} else {
				_, _ = fmt.Fprintf(out, "unset %s\n", variable.name)
			}
		}
	}
	if len(pathEntries) > 0 {
		if powershell {
			quoted := make([]string, len(pathEntries))
			for index, path := range pathEntries {
				quoted[index] = powershellQuote(path)
			}
			_, _ = fmt.Fprintf(out, "$env:Path = ((@(%s) + @($env:Path -split ';' | Where-Object { $_ })) | Select-Object -Unique) -join ';'\n", strings.Join(quoted, ", "))
		} else {
			_, _ = fmt.Fprintf(out, "export PATH=%q:$PATH\n", strings.Join(pathEntries, ":"))
		}
	}
	return nil
}

func writeEnvironmentVariable(out io.Writer, variable environmentVariable, powershell bool) {
	if powershell {
		_, _ = fmt.Fprintf(out, "$env:%s = %s\n", variable.name, powershellQuote(variable.value))
	} else {
		_, _ = fmt.Fprintf(out, "export %s=%q\n", variable.name, variable.value)
	}
}

func powershellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }

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
		path := stateStore.CurrentPath(tool)
		if runtime.GOOS == "windows" {
			installation, found := findInstallation(state.Installed[tool], version)
			if !found {
				return nil, fmt.Errorf("default %s version %s is not installed", tool, version)
			}
			path = installation.Path
		}
		targets[tool] = environmentTarget{Version: version, Path: path}
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

func parseEnvironmentFlags(args []string) (project, powershell bool, err error) {
	for _, arg := range args {
		switch arg {
		case "--project":
			project = true
		case "--powershell":
			powershell = true
		default:
			return false, false, errors.New("usage: sdk env [--project] [--powershell]")
		}
	}
	return project, powershell, nil
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
	case model.Gradle:
		return []environmentVariable{{name: "GRADLE_HOME", value: currentPath}}
	case model.Go:
		return []environmentVariable{{name: "GOROOT", value: currentPath}}
	case model.NodeJS:
		return []environmentVariable{{name: "NODE_HOME", value: currentPath}}
	default:
		return nil
	}
}
