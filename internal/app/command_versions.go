package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Denxuan/sdk/internal/installer"
	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/rustup"
	"github.com/Denxuan/sdk/internal/store"
)

func list(stateStore *store.Store, args []string, out io.Writer) error {
	if len(args) > 1 {
		return errors.New("usage: sdk list [tool]")
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	tools := model.Tools()
	if len(args) == 1 {
		tool, err := parseTool(args[0])
		if err != nil {
			return err
		}
		tools = []model.Tool{tool}
	}
	for _, tool := range tools {
		printInstalledVersions(out, tool, state.Installed[tool], state.Defaults[tool])
	}
	return nil
}

func printInstalledVersions(out io.Writer, tool model.Tool, installed []model.InstalledVersion, defaultVersion string) {
	_, _ = fmt.Fprintf(out, "%s", tool)
	if len(installed) == 0 {
		_, _ = fmt.Fprintln(out, ": no installed versions")
		return
	}
	_, _ = fmt.Fprintln(out, ":")
	sort.Slice(installed, func(i, j int) bool { return installed[i].Version > installed[j].Version })
	for _, item := range installed {
		marker := " "
		if item.Version == defaultVersion {
			marker = "*"
		}
		_, _ = fmt.Fprintf(out, "  %s %s  %s\n", marker, item.Version, item.Path)
	}
}

func current(stateStore *store.Store, args []string, out io.Writer) error {
	if len(args) > 1 {
		return errors.New("usage: sdk current [tool]")
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		printCurrentVersions(out, state)
		return nil
	}
	tool, err := parseTool(args[0])
	if err != nil {
		return err
	}
	if version := state.Defaults[tool]; version != "" {
		_, _ = fmt.Fprintln(out, version)
		return nil
	}
	return fmt.Errorf("no default %s version is set", tool)
}

func printCurrentVersions(out io.Writer, state model.State) {
	found := false
	for _, tool := range model.Tools() {
		if len(state.Installed[tool]) == 0 {
			continue
		}
		version := state.Defaults[tool]
		if version == "" {
			version = "not set"
		}
		_, _ = fmt.Fprintf(out, "%s: %s\n", tool, version)
		found = true
	}
	if !found {
		_, _ = fmt.Fprintln(out, "no tools are installed")
	}
}

func setDefault(ctx context.Context, stateStore *store.Store, args []string, out io.Writer, in io.Reader) error {
	if len(args) != 2 {
		return errors.New("usage: sdk default <tool> <version>")
	}
	tool, err := parseTool(args[0])
	if err != nil {
		return err
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	if !hasVersion(state.Installed[tool], args[1]) {
		args[1] = normalizeToolVersion(tool, args[1])
	}
	if !hasVersion(state.Installed[tool], args[1]) {
		return fmt.Errorf("%s %s is not installed", tool, args[1])
	}
	if tool == model.Rust {
		if err := rustup.Default(ctx, out, args[1]); err != nil {
			return err
		}
	}
	previousVersion := state.Defaults[tool]
	previousInstallation, hasPreviousInstallation := findInstallation(state.Installed[tool], previousVersion)
	state.Defaults[tool] = args[1]
	if err := stateStore.Save(state); err != nil {
		return err
	}
	installation, _ := findInstallation(state.Installed[tool], args[1])
	if tool == model.Java && installation.Managed {
		if err := installer.NormalizeJavaHome(installation.Path); err != nil {
			return fmt.Errorf("normalize Java installation: %w", err)
		}
	}
	if err := stateStore.SetCurrent(tool, installation.Path); err != nil {
		return fmt.Errorf("update current %s link: %w", tool, err)
	}
	_, _ = fmt.Fprintf(out, "default %s set to %s\n", tool, args[1])
	if hasPreviousInstallation && previousVersion != args[1] && supportsConfigurationMigration(tool) {
		migrate, err := askToMigrateConfiguration(out, in, tool, previousVersion, args[1])
		if err != nil {
			return err
		}
		if migrate {
			migrateToolConfiguration(ctx, tool, previousInstallation.Path, installation.Path, out)
		}
	}
	return nil
}

func supportsConfigurationMigration(tool model.Tool) bool {
	return tool == model.NodeJS || tool == model.Maven
}

func askToMigrateConfiguration(out io.Writer, in io.Reader, tool model.Tool, fromVersion, toVersion string) (bool, error) {
	_, _ = fmt.Fprintf(out, "Migrate %s configuration from %s to %s? [y/N]: ", tool, fromVersion, toVersion)
	answer, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && len(answer) == 0 {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	return isAffirmative(answer), nil
}

func uninstall(stateStore *store.Store, args []string, out io.Writer) error {
	if len(args) != 2 {
		return errors.New("usage: sdk uninstall <tool> <version>")
	}
	tool, err := parseTool(args[0])
	if err != nil {
		return err
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	version := args[1]
	isDefault := state.Defaults[tool] == version
	removed, remaining, found := removeVersion(state.Installed[tool], version)
	if !found {
		return fmt.Errorf("%s %s is not installed", tool, version)
	}
	if tool == model.Rust {
		if err := rustup.Uninstall(context.Background(), out, version); err != nil {
			return err
		}
	}
	if isDefault {
		if err := stateStore.RemoveCurrent(tool); err != nil {
			return err
		}
		delete(state.Defaults, tool)
	}
	if removed.Managed {
		if err := removeManagedFiles(stateStore, removed.Path); err != nil {
			return err
		}
	}
	state.Installed[tool] = remaining
	if err := stateStore.Save(state); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "uninstalled %s %s\n", tool, version)
	return nil
}

func removeVersion(installed []model.InstalledVersion, version string) (model.InstalledVersion, []model.InstalledVersion, bool) {
	remaining := make([]model.InstalledVersion, 0, len(installed))
	for index, item := range installed {
		if item.Version == version {
			remaining = append(remaining, installed[index+1:]...)
			return item, remaining, true
		}
		remaining = append(remaining, item)
	}
	return model.InstalledVersion{}, installed, false
}

func removeManagedFiles(stateStore *store.Store, path string) error {
	relativePath, err := filepath.Rel(stateStore.ToolsDir(), path)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("refusing to delete unmanaged path %s", path)
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove installed files: %w", err)
	}
	return nil
}

func hasVersion(installed []model.InstalledVersion, version string) bool {
	for _, item := range installed {
		if item.Version == version {
			return true
		}
	}
	return false
}

func findInstallation(installed []model.InstalledVersion, version string) (model.InstalledVersion, bool) {
	for _, item := range installed {
		if item.Version == version {
			return item, true
		}
	}
	return model.InstalledVersion{}, false
}
