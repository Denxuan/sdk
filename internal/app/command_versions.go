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
	"github.com/Denxuan/sdk/internal/shim"
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
	fmt.Fprintf(out, "%s", tool)
	if len(installed) == 0 {
		fmt.Fprintln(out, ": no installed versions")
		return
	}
	fmt.Fprintln(out, ":")
	sort.Slice(installed, func(i, j int) bool { return installed[i].Version > installed[j].Version })
	for _, item := range installed {
		marker := " "
		if item.Version == defaultVersion {
			marker = "*"
		}
		fmt.Fprintf(out, "  %s %s  %s\n", marker, item.Version, item.Path)
	}
}

func current(stateStore *store.Store, args []string, out io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: sdk current <tool>")
	}
	tool, err := parseTool(args[0])
	if err != nil {
		return err
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	if version := state.Defaults[tool]; version != "" {
		fmt.Fprintln(out, version)
		return nil
	}
	return fmt.Errorf("no default %s version is set", tool)
}

func setDefault(stateStore *store.Store, args []string, out io.Writer) error {
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
		return fmt.Errorf("%s %s is not installed", tool, args[1])
	}
	state.Defaults[tool] = args[1]
	if err := stateStore.Save(state); err != nil {
		return err
	}
	if binary, err := os.Executable(); err == nil {
		if err := shim.Ensure(stateStore.Home, binary); err != nil {
			return fmt.Errorf("create command shims: %w", err)
		}
	}
	fmt.Fprintf(out, "default %s set to %s\n", tool, args[1])
	return nil
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
	if state.Defaults[tool] == version {
		return fmt.Errorf("%s %s is the default; choose another version first", tool, version)
	}
	removed, remaining, found := removeVersion(state.Installed[tool], version)
	if !found {
		return fmt.Errorf("%s %s is not installed", tool, version)
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
	fmt.Fprintf(out, "uninstalled %s %s\n", tool, version)
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
