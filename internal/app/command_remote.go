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
	if tool == model.Java {
		formatJavaRemoteVersions(out, versions, state)
		return
	}
	printRemoteHeader(out, tool)
	printRemoteVersionRows(out, tool, versions, state)
	printRemoteLegend(out)
}

func formatJavaRemoteVersions(out io.Writer, versions []catalog.Version, state model.State) {
	var temurin, zulu []catalog.Version
	for _, version := range versions {
		if strings.HasSuffix(version.Number, "-zulu") {
			zulu = append(zulu, version)
		} else {
			temurin = append(temurin, version)
		}
	}
	printRemoteHeaderName(out, "Eclipse Temurin Java")
	printRemoteVersionRows(out, model.Java, temurin, state)
	printRemoteHeaderName(out, "Azul Zulu Java")
	printRemoteVersionRows(out, model.Java, zulu, state)
	printRemoteLegend(out)
}

func printRemoteVersionRows(out io.Writer, tool model.Tool, versions []catalog.Version, state model.State) {
	for index, version := range versions {
		_, _ = fmt.Fprintf(out, "%-*s", remoteColumnWidth, remoteVersionLabel(tool, version, state))
		if (index+1)%4 == 0 || index+1 == len(versions) {
			_, _ = fmt.Fprintln(out)
		}
	}
}

func printRemoteLegend(out io.Writer) {
	_, _ = fmt.Fprintln(out, strings.Repeat("=", 80))
	_, _ = fmt.Fprintln(out, "* - installed")
	_, _ = fmt.Fprintln(out, "> - currently in use")
	_, _ = fmt.Fprintln(out, strings.Repeat("=", 80))
}

func printRemoteHeader(out io.Writer, tool model.Tool) {
	printRemoteHeaderName(out, toolDisplayName(tool))
}

func printRemoteHeaderName(out io.Writer, name string) {
	_, _ = fmt.Fprintln(out, strings.Repeat("=", 80))
	_, _ = fmt.Fprintf(out, "Available %s Versions\n", name)
	_, _ = fmt.Fprintln(out, strings.Repeat("=", 80))
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
	case model.Gradle:
		return "Gradle"
	case model.Rust:
		return "Rust"
	case model.Go:
		return "Go"
	default:
		return string(tool)
	}
}
