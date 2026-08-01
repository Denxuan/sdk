package app

import (
	"fmt"
	"io"
)

func usage(out io.Writer) {
	fmt.Fprint(out, `sdk - developer tool version manager

Usage:
  sdk remote <tool>                    List official remote versions
  sdk available <tool>                 Alias of remote
  sdk install <tool> [version] [--path <directory>]
                                           Install a selected or recommended release
  sdk list [tool]                      List registered versions
  sdk current [tool]                   Print current versions, optionally for one tool
  sdk default <tool> <version>         Set the global default version
  sdk use <tool> <version>             Compatibility alias for default
  sdk uninstall <tool> <version>       Remove a non-default registration
  sdk env                              Print shell setup for managed commands
  sdk version                          Print the sdk version

Supported tools: java, nodejs, maven, go
State location: $SDK_HOME (or ~/.sdk)
`)
}
