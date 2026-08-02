package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	setupStart = "# >>> sdk initialize >>>"
	setupEnd   = "# <<< sdk initialize <<<"
)

func setupShell(args []string, out io.Writer) error {
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
	configPath := filepath.Join(home, ".zshrc")
	if err := addZshInitialization(configPath, binary); err != nil {
		return err
	}
	fmt.Fprintf(out, "Added sdk initialization to %s\n", configPath)
	fmt.Fprintf(out, "Run: source %s\n", configPath)
	return nil
}

func addZshInitialization(configPath, binary string) error {
	contents, err := os.ReadFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read zsh configuration: %w", err)
	}
	block := zshInitialization(binary)
	updated := replaceInitializationBlock(string(contents), block)
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("create zsh configuration directory: %w", err)
	}
	if err := os.WriteFile(configPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("write zsh configuration: %w", err)
	}
	return nil
}

func zshInitialization(binary string) string {
	return fmt.Sprintf(`%s
sdk() {
  %q "$@"
  local sdk_exit=$?

  case "$1" in
    default|use|install|update)
      if [ "$sdk_exit" -eq 0 ]; then
        eval "$(%q env)"
      fi
      ;;
  esac

  return "$sdk_exit"
}

eval "$(%q env)"
%s
`, setupStart, binary, binary, binary, setupEnd)
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
