package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Denxuan/sdk/internal/model"
)

func migrateToolConfiguration(ctx context.Context, tool model.Tool, sourcePath, destinationPath string, out io.Writer) {
	switch tool {
	case model.NodeJS:
		migrateNodeGlobalPackages(ctx, sourcePath, destinationPath, out)
	case model.Maven:
		migrateFiles(out, sourcePath, destinationPath, []string{
			filepath.Join("conf", "settings.xml"),
			filepath.Join("conf", "toolchains.xml"),
		})
	case model.Go:
		_, _ = fmt.Fprintln(out, "Go configuration is user-global and does not need migration.")
	case model.Java:
		_, _ = fmt.Fprintln(out, "Java truststore was kept from the new JDK; custom certificates are not overwritten automatically.")
	}
}

func migrateFiles(out io.Writer, sourcePath, destinationPath string, relativePaths []string) {
	migrated := 0
	for _, relativePath := range relativePaths {
		source := filepath.Join(sourcePath, relativePath)
		destination := filepath.Join(destinationPath, relativePath)
		data, err := os.ReadFile(source)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			_, _ = fmt.Fprintf(out, "Warning: could not read configuration %s: %v\n", source, err)
			continue
		}
		if _, err := os.Stat(destination); err == nil {
			backup := destination + ".sdk-default"
			if _, backupErr := os.Stat(backup); backupErr == nil {
				_, _ = fmt.Fprintf(out, "Warning: skipped %s because backup already exists: %s\n", relativePath, backup)
				continue
			}
			if err := os.Rename(destination, backup); err != nil {
				_, _ = fmt.Fprintf(out, "Warning: could not preserve new default %s: %v\n", relativePath, err)
				continue
			}
		} else if !os.IsNotExist(err) {
			_, _ = fmt.Fprintf(out, "Warning: could not inspect destination %s: %v\n", destination, err)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			_, _ = fmt.Fprintf(out, "Warning: could not create configuration directory for %s: %v\n", relativePath, err)
			continue
		}
		if err := os.WriteFile(destination, data, 0644); err != nil {
			_, _ = fmt.Fprintf(out, "Warning: could not migrate %s: %v\n", relativePath, err)
			continue
		}
		_, _ = fmt.Fprintf(out, "Migrated Maven configuration %s\n", relativePath)
		migrated++
	}
	if migrated == 0 {
		_, _ = fmt.Fprintln(out, "No Maven version-local configuration to migrate.")
	}
}

type npmGlobalPackage struct {
	Name    string
	Version string
}

func migrateNodeGlobalPackages(ctx context.Context, sourcePath, destinationPath string, out io.Writer) {
	sourceNPM := filepath.Join(sourcePath, "bin", "npm")
	destinationNPM := filepath.Join(destinationPath, "bin", "npm")
	packages, err := listNPMGlobalPackages(ctx, sourceNPM)
	if err != nil {
		_, _ = fmt.Fprintf(out, "Warning: could not list global npm packages from %s: %v\n", sourceNPM, err)
		return
	}
	if len(packages) == 0 {
		_, _ = fmt.Fprintln(out, "No third-party global npm packages to migrate.")
		return
	}
	_, _ = fmt.Fprintf(out, "Migrating %d global npm package(s)...\n", len(packages))
	for _, pkg := range packages {
		spec := npmPackageSpec(pkg)
		command := exec.CommandContext(ctx, destinationNPM, "install", "--global", spec)
		output, err := command.CombinedOutput()
		if err != nil {
			_, _ = fmt.Fprintf(out, "Warning: could not migrate %s: %v\n", spec, commandError(err, output))
			continue
		}
		_, _ = fmt.Fprintf(out, "Migrated %s\n", spec)
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
