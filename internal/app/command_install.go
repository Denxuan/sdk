package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Denxuan/sdk/internal/catalog"
	"github.com/Denxuan/sdk/internal/installer"
	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
)

func install(ctx context.Context, stateStore *store.Store, args []string, out io.Writer) error {
	toolName, version, existingPath, err := parseInstallArgs(args)
	if err != nil {
		return err
	}
	tool, err := parseTool(toolName)
	if err != nil {
		return err
	}
	if version == "" {
		if existingPath != "" {
			return errors.New("usage: sdk install <tool> <version> --path <directory>")
		}
		version, err = recommendedVersion(ctx, tool)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "selected %s %s\n", tool, version)
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	if hasVersion(state.Installed[tool], version) {
		return fmt.Errorf("%s %s is already registered", tool, version)
	}

	installPath := existingPath
	managed := false
	if installPath == "" {
		artifact, err := catalog.New().Artifact(tool, version)
		if err != nil {
			return err
		}
		installPath = filepath.Join(stateStore.ToolsDir(), string(tool), version)
		fmt.Fprintf(out, "downloading %s %s...\n", tool, version)
		if err := installer.New().Install(ctx, artifact.URL, installPath); err != nil {
			return err
		}
		managed = true
	} else if !isDirectory(installPath) {
		return fmt.Errorf("installation path must be a directory: %s", installPath)
	}

	state.Installed[tool] = append(state.Installed[tool], model.InstalledVersion{Version: version, Path: installPath, Managed: managed, InstalledAt: time.Now().UTC()})
	if err := stateStore.Save(state); err != nil {
		if managed {
			_ = os.RemoveAll(installPath)
		}
		return err
	}
	fmt.Fprintf(out, "installed %s %s at %s\n", tool, version, installPath)
	return nil
}

func parseInstallArgs(args []string) (tool, version, path string, err error) {
	positionals := make([]string, 0, 2)
	for index := 0; index < len(args); index++ {
		if args[index] != "--path" {
			positionals = append(positionals, args[index])
			continue
		}
		if index+1 == len(args) || path != "" {
			return "", "", "", errors.New("usage: sdk install <tool> <version> [--path <directory>]")
		}
		index++
		path = args[index]
	}
	if len(positionals) < 1 || len(positionals) > 2 {
		return "", "", "", errors.New("usage: sdk install <tool> [version] [--path <directory>]")
	}
	if len(positionals) == 2 {
		version = positionals[1]
	}
	return positionals[0], version, path, nil
}

func recommendedVersion(ctx context.Context, tool model.Tool) (string, error) {
	versions, err := catalog.New().Versions(ctx, tool)
	if err != nil {
		return "", err
	}
	if version := preferredVersion(versions); version != "" {
		return version, nil
	}
	return "", fmt.Errorf("no stable %s versions are available", tool)
}

func preferredVersion(versions []catalog.Version) string {
	for _, version := range versions {
		if version.LTS {
			return version.Number
		}
	}
	if len(versions) > 0 {
		return versions[0].Number
	}
	return ""
}

func isDirectory(path string) bool { info, err := os.Stat(path); return err == nil && info.IsDir() }
