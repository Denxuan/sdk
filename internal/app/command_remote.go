package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Denxuan/sdk/internal/catalog"
	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
)

const remoteColumnWidth = 20

func remote(ctx context.Context, stateStore *store.Store, args []string, out io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: sdk remote <tool>")
	}
	tool, err := parseTool(args[0])
	if err != nil {
		return err
	}
	versions, err := catalog.New().Versions(ctx, tool)
	if err != nil {
		return err
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	formatRemoteVersions(out, tool, versions, state)
	return nil
}

func formatRemoteVersions(out io.Writer, tool model.Tool, versions []catalog.Version, state model.State) {
	printRemoteHeader(out, tool)
	for index, version := range versions {
		fmt.Fprintf(out, "%-*s", remoteColumnWidth, remoteVersionLabel(tool, version, state))
		if (index+1)%4 == 0 || index+1 == len(versions) {
			fmt.Fprintln(out)
		}
	}
	fmt.Fprintln(out, strings.Repeat("=", 80))
	fmt.Fprintln(out, "* - installed")
	fmt.Fprintln(out, "> - currently in use")
	fmt.Fprintln(out, strings.Repeat("=", 80))
}

func printRemoteHeader(out io.Writer, tool model.Tool) {
	fmt.Fprintln(out, strings.Repeat("=", 80))
	fmt.Fprintf(out, "Available %s Versions\n", toolDisplayName(tool))
	fmt.Fprintln(out, strings.Repeat("=", 80))
}

func remoteVersionLabel(tool model.Tool, version catalog.Version, state model.State) string {
	label := version.Number
	if version.LTS {
		label += " LTS"
	}
	if state.Defaults[tool] == version.Number {
		return "> * " + label
	}
	if hasVersion(state.Installed[tool], version.Number) {
		return "  * " + label
	}
	return "    " + label
}

func toolDisplayName(tool model.Tool) string {
	switch tool {
	case model.Java:
		return "Java"
	case model.NodeJS:
		return "Node.js"
	case model.Maven:
		return "Maven"
	case model.MVND:
		return "Maven mvnd"
	case model.Go:
		return "Go"
	default:
		return string(tool)
	}
}
