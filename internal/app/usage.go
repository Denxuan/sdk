package app

import (
	"fmt"
	"io"
)

func usage(out io.Writer) {
	_, _ = fmt.Fprint(out, `sdk - developer tool version manager

Usage:
  sdk remote <tool>                    List official remote versions
  sdk available <tool>                 Alias of remote
  sdk install <tool> [version] [--path <directory>]
                                           Install a selected or recommended release
  sdk update [tool] [--check]             Update managed tools, or only check available updates
  sdk list [tool]                      List registered versions
  sdk current [tool]                   Print current versions, optionally for one tool
  sdk default <tool> <version>         Set the global default version
  sdk use <tool> <version>             Compatibility alias for default
  sdk uninstall <tool> <version>       Remove a non-default registration
  sdk env [--project]                  Print environment variables; project versions override defaults
  sdk doctor                           Diagnose installations, current links, and environment
  sdk path <tool>                      Print the current tool directory
  sdk which <tool>                     Print the current tool executable path
  sdk project init                     Create .sdk-version from current defaults
  sdk project set <tool> <version>     Set a version in .sdk-version in the current directory
  sdk project list                     Show the nearest .sdk-version file
  sdk setup zsh                        Add sdk initialization to ~/.zshrc
  sdk mcp serve                        Run the MCP server over stdio
  sdk selfupdate [version]             Update sdk from a GitHub Release
  sdk version                          Print the sdk version

Supported tools: java, nodejs, maven, mvnd, gradle, go
State location: $SDK_HOME (or ~/.sdk)
`)
}
