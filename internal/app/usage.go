package app

import (
	"fmt"
	"io"
)

func usage(out io.Writer) {
	_, _ = fmt.Fprint(out, `sdk - developer tool version manager

Usage:
  sdk remote <tool> [--refresh]        List official remote versions
  sdk available <tool> [--refresh]     Alias of remote
  sdk install <tool> [version] [--path <directory>]
                                           Install a selected or recommended release
  sdk update [tool] [--check]             Update managed tools, or only check available updates
  sdk list [tool]                      List registered versions
  sdk current [tool]                   Print current versions, optionally for one tool
  sdk default <tool> <version>         Set the global default version
  sdk use <tool> <version>             Compatibility alias for default
  sdk uninstall <tool> <version>       Uninstall a version and clear it if it is default
  sdk uninstall rustup                  Remove rustup and all Rust toolchains
  sdk env [--project] [--powershell]   Print shell environment; project versions override defaults
  sdk doctor                           Diagnose installations, current links, and environment
  sdk clean                            Remove incomplete SDK tool downloads
  sdk path <tool>                      Print the current tool directory
  sdk which <tool>                     Print the current tool executable path
  sdk project init                     Create .sdk-version from current defaults
  sdk project set <tool> <version>     Set a version in .sdk-version in the current directory
  sdk project install                  Install tool versions pinned in the nearest .sdk-version
  sdk project status                   Check pinned versions and their installations
  sdk project unset <tool>             Remove a tool version from .sdk-version in the current directory
  sdk project remove                   Remove the nearest .sdk-version file
  sdk project list                     Show the nearest .sdk-version file
  sdk setup <bash|zsh|powershell>      Add sdk initialization to the shell profile
  sdk mcp serve                        Run the MCP server over stdio
  sdk rustup <args...>                 Run rustup with automatic installation
  sdk selfupdate [version]             Update sdk from a GitHub Release
  sdk version                          Print the sdk version

Supported tools: java, nodejs, maven, mvnd, gradle, rust, go
State location: $SDK_HOME (or ~/.sdk)
`)
}
