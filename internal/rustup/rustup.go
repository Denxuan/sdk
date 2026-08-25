package rustup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func Ensure(ctx context.Context, out io.Writer) (string, error) {
	if path := findPath(); path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".cargo", "bin", "rustup")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	target, err := targetTriple()
	if err != nil {
		return "", err
	}
	url := "https://static.rust-lang.org/rustup/dist/" + target + "/rustup-init"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		return "", fmt.Errorf("download rustup-init: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download rustup-init: server returned %s", response.Status)
	}
	temp, err := os.CreateTemp("", "rustup-init-")
	if err != nil {
		return "", err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := io.Copy(temp, response.Body); err != nil {
		temp.Close()
		return "", err
	}
	if err := temp.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(tempPath, 0755); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, tempPath, "-y", "--no-modify-path")
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("install rustup: %w", err)
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("rustup was installed but not found at %s: %w", path, err)
	}
	return path, nil
}

func findPath() string {
	if path, err := exec.LookPath("rustup"); err == nil {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, ".cargo", "bin", "rustup")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func Run(ctx context.Context, out io.Writer, args ...string) error {
	path, err := Ensure(ctx, out)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdout, cmd.Stderr = out, out
	return cmd.Run()
}

func Install(ctx context.Context, out io.Writer, version string) (string, error) {
	if err := Run(ctx, out, "toolchain", "install", version); err != nil {
		return "", err
	}
	return ToolchainPath(ctx, out, version)
}

func Default(ctx context.Context, out io.Writer, version string) error {
	return Run(ctx, out, "default", version)
}

func Uninstall(ctx context.Context, out io.Writer, version string) error {
	return Run(ctx, out, "toolchain", "uninstall", version)
}

func Update(ctx context.Context, out io.Writer) error { return Run(ctx, out, "update") }

func SelfUninstall(ctx context.Context, out io.Writer) error {
	path := findPath()
	if path == "" {
		return fmt.Errorf("rustup is not installed")
	}
	cmd := exec.CommandContext(ctx, path, "self", "uninstall")
	cmd.Stdin = strings.NewReader("y\n")
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("uninstall rustup: %w", err)
	}
	return nil
}

func ToolchainPath(ctx context.Context, out io.Writer, version string) (string, error) {
	path, err := Ensure(ctx, out)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, path, "which", "rustc", "--toolchain", version)
	var result bytes.Buffer
	cmd.Stdout, cmd.Stderr = &result, out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("find Rust toolchain path: %w", err)
	}
	bin := strings.TrimSpace(result.String())
	if bin == "" {
		return "", fmt.Errorf("rustup returned an empty Rust toolchain path")
	}
	return filepath.Dir(filepath.Dir(bin)), nil
}

func CargoBin() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cargo", "bin")
}

func targetTriple() (string, error) {
	osName := map[string]string{"darwin": "apple-darwin", "linux": "unknown-linux-gnu", "windows": "pc-windows-msvc"}[runtime.GOOS]
	arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[runtime.GOARCH]
	if osName == "" || arch == "" {
		return "", fmt.Errorf("rustup is not supported on %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return arch + "-" + osName, nil
}
