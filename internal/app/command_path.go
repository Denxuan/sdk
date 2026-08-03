package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
)

func printToolPath(stateStore *store.Store, args []string, out io.Writer) error {
	tool, err := currentTool(args, stateStore)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, stateStore.CurrentPath(tool))
	return nil
}

func printToolExecutable(stateStore *store.Store, args []string, out io.Writer) error {
	tool, err := currentTool(args, stateStore)
	if err != nil {
		return err
	}
	executable := filepath.Join(stateStore.CurrentPath(tool), "bin", executableName(tool))
	info, err := os.Stat(executable)
	if err != nil {
		return fmt.Errorf("current %s executable is unavailable at %s: %w", tool, executable, err)
	}
	if info.IsDir() || info.Mode()&0111 == 0 {
		return fmt.Errorf("current %s executable is not executable: %s", tool, executable)
	}
	fmt.Fprintln(out, executable)
	return nil
}

func currentTool(args []string, stateStore *store.Store) (model.Tool, error) {
	if len(args) != 1 {
		return "", errors.New("usage: sdk <path|which> <tool>")
	}
	tool, err := parseTool(args[0])
	if err != nil {
		return "", err
	}
	state, err := stateStore.Load()
	if err != nil {
		return "", err
	}
	if state.Defaults[tool] == "" {
		return "", fmt.Errorf("no default %s version is set", tool)
	}
	return tool, nil
}
