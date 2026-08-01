package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
)

func update(ctx context.Context, stateStore *store.Store, args []string, out io.Writer) error {
	if len(args) > 1 {
		return errors.New("usage: sdk update [tool]")
	}

	targets, err := updateTargets(stateStore, args)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		fmt.Fprintln(out, "no managed tools to update")
		return nil
	}

	for _, tool := range targets {
		if err := updateTool(ctx, stateStore, tool, out); err != nil {
			return err
		}
	}
	return nil
}

func updateTargets(stateStore *store.Store, args []string) ([]model.Tool, error) {
	if len(args) == 1 {
		tool, err := parseTool(args[0])
		if err != nil {
			return nil, err
		}
		return []model.Tool{tool}, nil
	}

	state, err := stateStore.Load()
	if err != nil {
		return nil, err
	}
	targets := make([]model.Tool, 0, len(model.Tools()))
	for _, tool := range model.Tools() {
		if len(state.Installed[tool]) > 0 {
			targets = append(targets, tool)
		}
	}
	return targets, nil
}

func updateTool(ctx context.Context, stateStore *store.Store, tool model.Tool, out io.Writer) error {
	version, err := recommendedVersion(ctx, tool)
	if err != nil {
		return err
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	if hasVersion(state.Installed[tool], version) {
		fmt.Fprintf(out, "%s is already up to date (%s)\n", tool, version)
		return nil
	}
	previousVersions := state.Installed[tool]
	fmt.Fprintf(out, "updating %s to %s\n", tool, version)
	if err := install(ctx, stateStore, []string{string(tool), version}, out); err != nil {
		return err
	}
	return offerOldVersionCleanup(stateStore, tool, version, previousVersions, out, os.Stdin)
}

func offerOldVersionCleanup(stateStore *store.Store, tool model.Tool, newVersion string, previous []model.InstalledVersion, out io.Writer, in io.Reader) error {
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	candidates := oldManagedVersions(previous, newVersion, state.Defaults[tool])
	if len(candidates) == 0 {
		return nil
	}
	versions := make([]string, 0, len(candidates))
	for _, item := range candidates {
		versions = append(versions, item.Version)
	}
	fmt.Fprintf(out, "Remove old %s versions (%s)? [y/N]: ", tool, strings.Join(versions, ", "))
	answer, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && len(answer) == 0 {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	if !isAffirmative(answer) {
		return nil
	}

	for _, item := range candidates {
		if err := removeManagedFiles(stateStore, item.Path); err != nil {
			return err
		}
		_, remaining, _ := removeVersion(state.Installed[tool], item.Version)
		state.Installed[tool] = remaining
	}
	if err := stateStore.Save(state); err != nil {
		return err
	}
	fmt.Fprintf(out, "removed old %s versions: %s\n", tool, strings.Join(versions, ", "))
	return nil
}

func oldManagedVersions(installed []model.InstalledVersion, newVersion, defaultVersion string) []model.InstalledVersion {
	candidates := make([]model.InstalledVersion, 0, len(installed))
	for _, item := range installed {
		if item.Managed && item.Version != newVersion && item.Version != defaultVersion {
			candidates = append(candidates, item)
		}
	}
	return candidates
}
