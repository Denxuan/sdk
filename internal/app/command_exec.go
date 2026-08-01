package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"

	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/shim"
	"github.com/Denxuan/sdk/internal/store"
)

func printEnvironment(stateStore *store.Store, out io.Writer) error {
	_, err := fmt.Fprintf(out, "export PATH=%q:$PATH\n", filepath.Join(stateStore.Home, "shims"))
	return err
}

func execute(stateStore *store.Store, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: sdk exec <command> [arguments...]")
	}
	tool, supported := shim.ToolFor(args[0])
	if !supported {
		return fmt.Errorf("unsupported managed command %q", args[0])
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	version := state.Defaults[tool]
	if version == "" {
		return fmt.Errorf("no default %s version is set", tool)
	}
	installation, found := findInstallation(state.Installed[tool], version)
	if !found {
		return fmt.Errorf("default %s %s is not installed", tool, version)
	}
	binary, err := executablePath(installation.Path, args[0])
	if err != nil {
		return err
	}
	command := osexec.Command(binary, args[1:]...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	return command.Run()
}

func findInstallation(installed []model.InstalledVersion, version string) (model.InstalledVersion, bool) {
	for _, item := range installed {
		if item.Version == version {
			return item, true
		}
	}
	return model.InstalledVersion{}, false
}

func executablePath(root, command string) (string, error) {
	for _, directory := range executableDirectories(root) {
		for _, filename := range executableNames(command) {
			path := filepath.Join(directory, filename)
			if isExecutableFile(path) {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("%s executable was not found in %s", command, root)
}

func executableDirectories(root string) []string {
	return []string{filepath.Join(root, "bin"), filepath.Join(root, "Contents", "Home", "bin")}
}

func executableNames(command string) []string {
	if runtime.GOOS == "windows" {
		return []string{command, command + ".exe", command + ".cmd"}
	}
	return []string{command}
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
