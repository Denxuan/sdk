package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type npmGlobalPackage struct {
	Name    string
	Version string
}

func migrateNodeGlobalPackages(ctx context.Context, sourcePath, destinationPath string, out io.Writer) {
	sourceNPM := filepath.Join(sourcePath, "bin", "npm")
	destinationNPM := filepath.Join(destinationPath, "bin", "npm")
	packages, err := listNPMGlobalPackages(ctx, sourceNPM)
	if err != nil {
		fmt.Fprintf(out, "Warning: could not list global npm packages from %s: %v\n", sourceNPM, err)
		return
	}
	if len(packages) == 0 {
		fmt.Fprintln(out, "No third-party global npm packages to migrate.")
		return
	}
	fmt.Fprintf(out, "Migrating %d global npm package(s)...\n", len(packages))
	for _, pkg := range packages {
		spec := npmPackageSpec(pkg)
		command := exec.CommandContext(ctx, destinationNPM, "install", "--global", spec)
		output, err := command.CombinedOutput()
		if err != nil {
			fmt.Fprintf(out, "Warning: could not migrate %s: %v\n", spec, commandError(err, output))
			continue
		}
		fmt.Fprintf(out, "Migrated %s\n", spec)
	}
}

func listNPMGlobalPackages(ctx context.Context, npm string) ([]npmGlobalPackage, error) {
	command := exec.CommandContext(ctx, npm, "ls", "--global", "--depth=0", "--json")
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, commandError(err, output)
	}
	return parseNPMGlobalPackages(output)
}

func parseNPMGlobalPackages(data []byte) ([]npmGlobalPackage, error) {
	var response struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse npm global package list: %w", err)
	}
	packages := make([]npmGlobalPackage, 0, len(response.Dependencies))
	for name, dependency := range response.Dependencies {
		if name == "npm" || name == "corepack" || dependency.Version == "" {
			continue
		}
		packages = append(packages, npmGlobalPackage{Name: name, Version: dependency.Version})
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].Name < packages[j].Name })
	return packages, nil
}

func npmPackageSpec(pkg npmGlobalPackage) string {
	return pkg.Name + "@" + pkg.Version
}

func commandError(err error, output []byte) error {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, message)
}
