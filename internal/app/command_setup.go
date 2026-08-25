package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Denxuan/sdk/internal/store"
)

const (
	setupStart  = "# >>> sdk initialize >>>"
	setupEnd    = "# <<< sdk initialize <<<"
	zshInitFile = "init.zsh"
)

func setupShell(stateStore *store.Store, args []string, out io.Writer) error {
	if len(args) != 1 || args[0] != "zsh" {
		return errors.New("usage: sdk setup zsh")
	}
	binary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find sdk executable: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find home directory: %w", err)
	}
	sdkHome, err := filepath.Abs(stateStore.Home)
	if err != nil {
		return fmt.Errorf("resolve SDK home: %w", err)
	}
	initPath := filepath.Join(sdkHome, zshInitFile)
	if err := writeZshInitScript(initPath, binary); err != nil {
		return err
	}
	configPath := filepath.Join(home, ".zshrc")
	if err := addZshInitialization(configPath, sdkHome); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "Created sdk shell script at %s\n", initPath)
	_, _ = fmt.Fprintf(out, "Added sdk initialization to %s\n", configPath)
	_, _ = fmt.Fprintf(out, "Run: source %s\n", configPath)
	return nil
}

func writeZshInitScript(path, binary string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create SDK home: %w", err)
	}
	if err := os.WriteFile(path, []byte(zshScript(binary)), 0644); err != nil {
		return fmt.Errorf("write zsh initialization script: %w", err)
	}
	return nil
}

func addZshInitialization(configPath, sdkHome string) error {
	contents, err := os.ReadFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read zsh configuration: %w", err)
	}
	block := zshInitialization(sdkHome)
	updated := replaceInitializationBlock(string(contents), block)
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("create zsh configuration directory: %w", err)
	}
	if err := os.WriteFile(configPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("write zsh configuration: %w", err)
	}
	return nil
}

func zshInitialization(sdkHome string) string {
	return fmt.Sprintf(`%s
export SDK_HOME=%q
if [ -r "$SDK_HOME/%s" ]; then
  source "$SDK_HOME/%s"
fi
%s
`, setupStart, sdkHome, zshInitFile, zshInitFile, setupEnd)
}

func zshScript(binary string) string {
	return fmt.Sprintf(`typeset -U path

_sdk_refresh_environment() {
  eval "$(%q env --project)"
}

autoload -Uz add-zsh-hook
add-zsh-hook -d chpwd _sdk_refresh_environment 2>/dev/null
add-zsh-hook chpwd _sdk_refresh_environment

sdk() {
  %q "$@"
  local sdk_exit=$?

  case "$1" in
    default|use|install|update|uninstall)
      if [ "$sdk_exit" -eq 0 ]; then
        _sdk_refresh_environment
      fi
      ;;
  esac

  return "$sdk_exit"
}

_sdk_refresh_environment
`, binary, binary)
}

func replaceInitializationBlock(contents, block string) string {
	start := strings.Index(contents, setupStart)
	if start >= 0 {
		end := strings.Index(contents[start:], setupEnd)
		if end >= 0 {
			end += start + len(setupEnd)
			contents = contents[:start] + contents[end:]
		}
	}
	contents = strings.TrimRight(contents, "\n")
	if contents == "" {
		return block
	}
	return contents + "\n\n" + block
}
