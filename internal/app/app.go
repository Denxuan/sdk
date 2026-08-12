package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Denxuan/sdk/internal/buildinfo"
	mcpserver "github.com/Denxuan/sdk/internal/mcp"
	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
)

func Run(ctx context.Context, args []string, out, errOut io.Writer) error {
	if len(args) == 0 {
		usage(out)
		return nil
	}
	home, err := resolveHome()
	if err != nil {
		return err
	}
	stateStore := store.New(home)

	switch args[0] {
	case "help", "--help", "-h":
		usage(out)
		return nil
	case "version":
		fmt.Fprintf(out, "sdk %s\n", buildinfo.Version)
		return nil
	case "selfupdate":
		return selfUpdate(ctx, args[1:], out)
	case "env":
		return printEnvironment(stateStore, args[1:], out)
	case "doctor":
		return doctor(stateStore, args[1:], out)
	case "path":
		return printToolPath(stateStore, args[1:], out)
	case "which":
		return printToolExecutable(stateStore, args[1:], out)
	case "project":
		return project(stateStore, args[1:], out)
	case "setup":
		return setupShell(stateStore, args[1:], out)
	case "list":
		return list(stateStore, args[1:], out)
	case "current":
		return current(stateStore, args[1:], out)
	case "use", "default":
		return setDefault(ctx, stateStore, args[1:], out, os.Stdin)
	case "uninstall":
		return uninstall(stateStore, args[1:], out)
	case "install":
		return install(ctx, stateStore, args[1:], out)
	case "update":
		return update(ctx, stateStore, args[1:], out)
	case "remote", "available":
		return remote(ctx, stateStore, args[1:], out)
	case "mcp":
		if len(args) != 2 || args[1] != "serve" {
			return fmt.Errorf("usage: sdk mcp serve")
		}
		return mcpserver.NewServer(home).Run(ctx)
	default:
		return fmt.Errorf("unknown command %q (run sdk help)", args[0])
	}
}

func resolveHome() (string, error) {
	if home := os.Getenv("SDK_HOME"); home != "" {
		return home, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".sdk"), nil
}

func parseTool(value string) (model.Tool, error) {
	tool := model.Tool(strings.ToLower(value))
	if !tool.Valid() {
		return "", fmt.Errorf("unsupported tool %q; supported: java, nodejs, maven, go", value)
	}
	return tool, nil
}
