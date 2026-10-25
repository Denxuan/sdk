package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Denxuan/sdk/internal/store"
)

const (
	setupStart   = "# >>> sdk initialize >>>"
	setupEnd     = "# <<< sdk initialize <<<"
	bashInitFile = "init.bash"
	psInitFile   = "init.ps1"
	zshInitFile  = "init.zsh"
)

func setupShell(stateStore *store.Store, args []string, out io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: sdk setup <bash|zsh|powershell>")
	}
	shell := args[0]
	if runtime.GOOS == "windows" && shell != "powershell" || runtime.GOOS != "windows" && shell != "bash" && shell != "zsh" {
		return errors.New("sdk setup supports Bash/Zsh on macOS/Linux and PowerShell on Windows")
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
	if shell == "powershell" {
		return setupPowerShell(sdkHome, home, binary, out)
	}
	configName, initName := "."+shell+"rc", shellInitFile(shell)
	initPath := filepath.Join(sdkHome, initName)
	if err := writeShellInitScript(initPath, shell, binary); err != nil {
		return err
	}
	configPath := filepath.Join(home, configName)
	if err := addShellInitialization(configPath, sdkHome, shell); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "Created sdk shell script at %s\n", initPath)
	_, _ = fmt.Fprintf(out, "Added sdk initialization to %s\n", configPath)
	_, _ = fmt.Fprintf(out, "Run: source %s\n", configPath)
	return nil
}

func shellInitFile(shell string) string {
	if shell == "bash" {
		return bashInitFile
	}
	if shell == "powershell" {
		return psInitFile
	}
	return zshInitFile
}

func writeShellInitScript(path, shell, binary string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create SDK home: %w", err)
	}
	script := zshScript(binary)
	if shell == "bash" {
		script = bashScript(binary)
	} else if shell == "powershell" {
		script = powershellScript(binary)
	}
	if err := os.WriteFile(path, []byte(script), 0644); err != nil {
		return fmt.Errorf("write %s initialization script: %w", shell, err)
	}
	return nil
}

func addShellInitialization(configPath, sdkHome, shell string) error {
	contents, err := os.ReadFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read %s configuration: %w", shell, err)
	}
	block := zshInitialization(sdkHome)
	if shell == "bash" {
		block = bashInitialization(sdkHome)
	} else if shell == "powershell" {
		block = powershellInitialization(sdkHome)
	}
	updated := replaceInitializationBlock(string(contents), block)
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	if err := os.WriteFile(configPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("write %s configuration: %w", shell, err)
	}
	return nil
}

func setupPowerShell(sdkHome, home, binary string, out io.Writer) error {
	initPath := filepath.Join(sdkHome, psInitFile)
	if err := writeShellInitScript(initPath, "powershell", binary); err != nil {
		return err
	}
	profiles := []string{
		filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
	}
	for _, profile := range profiles {
		if err := addShellInitialization(profile, sdkHome, "powershell"); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "Added sdk initialization to %s\n", profile)
	}
	_, _ = fmt.Fprintf(out, "Created sdk shell script at %s\n", initPath)
	_, _ = fmt.Fprintln(out, "Restart PowerShell to apply the environment.")
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
    project)
      case "$2" in
        set|unset|remove|install)
          if [ "$sdk_exit" -eq 0 ]; then
            _sdk_refresh_environment
          fi
          ;;
      esac
      ;;
  esac

  return "$sdk_exit"
}

_sdk_refresh_environment
`, binary, binary)
}

func bashInitialization(sdkHome string) string {
	return fmt.Sprintf(`%s
export SDK_HOME=%q
if [ -r "$SDK_HOME/%s" ]; then
  source "$SDK_HOME/%s"
fi
%s
`, setupStart, sdkHome, bashInitFile, bashInitFile, setupEnd)
}

func powershellInitialization(sdkHome string) string {
	return fmt.Sprintf("%s\n$env:SDK_HOME = %s\n. (Join-Path $env:SDK_HOME %s)\n%s\n",
		setupStart, powershellQuote(sdkHome), powershellQuote(psInitFile), setupEnd)
}

func bashScript(binary string) string {
	return fmt.Sprintf(`_sdk_refresh_environment() {
  eval "$(%q env --project)"
}

_sdk_check_directory() {
  if [[ "$PWD" != "${_SDK_LAST_PWD-}" ]]; then
    _SDK_LAST_PWD=$PWD
    _sdk_refresh_environment
  fi
}

case ";${PROMPT_COMMAND-};" in
  *";_sdk_check_directory;"*) ;;
  *) PROMPT_COMMAND="_sdk_check_directory${PROMPT_COMMAND:+;$PROMPT_COMMAND}" ;;
esac

sdk() {
  %q "$@"
  local sdk_exit=$?

  case "$1" in
    default|use|install|update|uninstall)
      if [ "$sdk_exit" -eq 0 ]; then
        _sdk_refresh_environment
      fi
      ;;
    project)
      case "$2" in
        set|unset|remove|install)
          if [ "$sdk_exit" -eq 0 ]; then
            _sdk_refresh_environment
          fi
          ;;
      esac
      ;;
  esac

  return "$sdk_exit"
}

_SDK_LAST_PWD=$PWD
_sdk_refresh_environment
`, binary, binary)
}

func powershellScript(binary string) string {
	return fmt.Sprintf(`$global:__sdkExecutable = %s
function global:Update-SdkEnvironment {
  $output = & $global:__sdkExecutable env --powershell --project
  if ($LASTEXITCODE -eq 0) {
    Invoke-Expression ($output -join [Environment]::NewLine)
  }
}

function global:sdk {
  $sdkArgs = @($args)
  & $global:__sdkExecutable @sdkArgs
  $sdkExit = $LASTEXITCODE
  $refresh = $false
  if ($sdkArgs.Count -gt 0) {
    switch ($sdkArgs[0]) {
      { $_ -in @('default', 'use', 'install', 'update', 'uninstall') } { $refresh = $true }
      'project' {
        if ($sdkArgs.Count -gt 1 -and $sdkArgs[1] -in @('set', 'unset', 'remove', 'install')) { $refresh = $true }
      }
    }
  }
  if ($sdkExit -eq 0 -and $refresh) { Update-SdkEnvironment }
  $global:LASTEXITCODE = $sdkExit
}

if (-not $global:__sdkPromptInstalled) {
  $global:__sdkOriginalPrompt = (Get-Command prompt -CommandType Function -ErrorAction SilentlyContinue).ScriptBlock
  function global:prompt {
    $currentLocation = (Get-Location).Path
    if ($global:__sdkLastLocation -ne $currentLocation) {
      $global:__sdkLastLocation = $currentLocation
      Update-SdkEnvironment
    }
    if ($global:__sdkOriginalPrompt) { & $global:__sdkOriginalPrompt }
    else { "PS $currentLocation> " }
  }
  $global:__sdkPromptInstalled = $true
}

$global:__sdkLastLocation = (Get-Location).Path
Update-SdkEnvironment
`, powershellQuote(binary))
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
