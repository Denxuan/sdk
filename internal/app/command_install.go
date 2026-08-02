package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
		progress := newDownloadProgress(out, tool, version)
		err = installer.New().WithProgress(progress.Update).WithRetry(progress.Retry).Install(ctx, artifact.URL, installPath)
		progress.Finish()
		if err != nil {
			return err
		}
		managed = true
		if tool == model.Java {
			if err := installer.NormalizeJavaHome(installPath); err != nil {
				_ = os.RemoveAll(installPath)
				return err
			}
		}
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
	if managed {
		setAsDefault, err := askToSetDefault(out, os.Stdin, tool, version)
		if err != nil {
			return err
		}
		if setAsDefault {
			return setDefault(stateStore, []string{string(tool), version}, out)
		}
	}
	return nil
}

func askToSetDefault(out io.Writer, in io.Reader, tool model.Tool, version string) (bool, error) {
	fmt.Fprintf(out, "Set %s %s as default? [y/N]: ", tool, version)
	answer, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && len(answer) == 0 {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	return isAffirmative(answer), nil
}

func isAffirmative(answer string) bool {
	normalized := strings.ToLower(strings.TrimSpace(answer))
	return normalized == "y" || normalized == "yes"
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
