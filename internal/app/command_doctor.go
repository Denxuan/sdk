package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
)

func doctor(stateStore *store.Store, args []string, out io.Writer) error {
	if len(args) != 0 {
		return errors.New("usage: sdk doctor")
	}

	report := &doctorReport{out: out}
	_, _ = fmt.Fprintln(out, "SDK doctor")
	report.checkDirectory("SDK home", stateStore.Home)

	state, err := stateStore.Load()
	if err != nil {
		report.error("state file", err.Error())
		return report.result()
	}
	if _, err := os.Stat(stateStore.StatePath()); err == nil {
		report.ok("state file", stateStore.StatePath())
	} else if errors.Is(err, os.ErrNotExist) {
		report.warn("state file", "not created yet; install a tool to initialize it")
	} else {
		report.error("state file", err.Error())
	}

	installedCount := 0
	for _, tool := range model.Tools() {
		installed := append([]model.InstalledVersion(nil), state.Installed[tool]...)
		sort.Slice(installed, func(i, j int) bool { return installed[i].Version < installed[j].Version })
		for _, installation := range installed {
			installedCount++
			report.checkDirectory(fmt.Sprintf("%s %s", tool, installation.Version), installation.Path)
		}
	}
	if installedCount == 0 {
		report.warn("installations", "no managed tools are installed")
	}

	for _, tool := range model.Tools() {
		version := state.Defaults[tool]
		if version == "" {
			continue
		}
		installation, found := findInstallation(state.Installed[tool], version)
		if !found {
			report.error(fmt.Sprintf("%s current", tool), fmt.Sprintf("default version %s is not registered", version))
			continue
		}
		checkCurrentLink(report, stateStore, tool, installation)
		checkEnvironment(report, stateStore, tool)
	}

	report.summary()
	return report.result()
}

func checkCurrentLink(report *doctorReport, stateStore *store.Store, tool model.Tool, installation model.InstalledVersion) {
	currentPath := stateStore.CurrentPath(tool)
	info, err := os.Lstat(currentPath)
	if errors.Is(err, os.ErrNotExist) {
		report.error(fmt.Sprintf("%s current", tool), "link does not exist")
		return
	}
	if err != nil {
		report.error(fmt.Sprintf("%s current", tool), err.Error())
		return
	}
	if info.Mode()&os.ModeSymlink == 0 {
		report.error(fmt.Sprintf("%s current", tool), "path exists but is not a symbolic link")
		return
	}

	target, err := filepath.EvalSymlinks(currentPath)
	if err != nil {
		report.error(fmt.Sprintf("%s current", tool), fmt.Sprintf("broken link: %v", err))
		return
	}
	expected, err := filepath.EvalSymlinks(installation.Path)
	if err != nil {
		expected, err = filepath.Abs(installation.Path)
		if err != nil {
			report.error(fmt.Sprintf("%s current", tool), err.Error())
			return
		}
	}
	if filepath.Clean(target) != filepath.Clean(expected) {
		report.error(fmt.Sprintf("%s current", tool), fmt.Sprintf("points to %s, expected %s", target, expected))
		return
	}
	report.ok(fmt.Sprintf("%s current", tool), currentPath)

	for _, name := range executableNames(tool) {
		executable := filepath.Join(currentPath, "bin", name)
		info, err = os.Stat(executable)
		if err != nil {
			report.error(fmt.Sprintf("%s executable %s", tool, name), fmt.Sprintf("%s: %v", executable, err))
			continue
		}
		if info.IsDir() || info.Mode()&0111 == 0 {
			report.error(fmt.Sprintf("%s executable %s", tool, name), fmt.Sprintf("%s is not executable", executable))
			continue
		}
		report.ok(fmt.Sprintf("%s executable %s", tool, name), executable)
	}
}

func checkEnvironment(report *doctorReport, stateStore *store.Store, tool model.Tool) {
	currentPath := stateStore.CurrentPath(tool)
	expectedBin := filepath.Join(currentPath, "bin")
	if pathContains(os.Getenv("PATH"), expectedBin) {
		report.ok(fmt.Sprintf("%s PATH", tool), expectedBin)
	} else {
		report.warn(fmt.Sprintf("%s PATH", tool), fmt.Sprintf("%s is missing; run eval \"$(sdk env)\"", expectedBin))
	}
	if tool == model.Go {
		for _, goPath := range goToolPaths() {
			if pathContains(os.Getenv("PATH"), goPath) {
				report.ok("go tools PATH", goPath)
			} else {
				report.warn("go tools PATH", fmt.Sprintf("%s is missing; run eval \"$(sdk env)\"", goPath))
			}
		}
	}

	for _, variable := range environmentVariables(tool, currentPath) {
		if os.Getenv(variable.name) == variable.value {
			report.ok(variable.name, variable.value)
		} else {
			report.warn(variable.name, fmt.Sprintf("expected %s; run eval \"$(sdk env)\"", variable.value))
		}
	}
}

func executableName(tool model.Tool) string {
	return executableNames(tool)[0]
}

func executableNames(tool model.Tool) []string {
	switch tool {
	case model.Java:
		return []string{"java"}
	case model.Maven:
		return []string{"mvn"}
	case model.MVND:
		return []string{"mvnd"}
	case model.Gradle:
		return []string{"gradle"}
	case model.Go:
		return []string{"go"}
	case model.NodeJS:
		return []string{"node", "npm", "npx"}
	default:
		return []string{string(tool)}
	}
}

func pathContains(pathValue, expected string) bool {
	expected = filepath.Clean(expected)
	for _, entry := range filepath.SplitList(pathValue) {
		if filepath.Clean(entry) == expected {
			return true
		}
	}
	return false
}

type doctorReport struct {
	out      io.Writer
	passed   int
	warnings int
	errors   int
}

func (r *doctorReport) ok(name, detail string) {
	r.passed++
	_, _ = fmt.Fprintf(r.out, "[OK]   %s: %s\n", name, detail)
}

func (r *doctorReport) warn(name, detail string) {
	r.warnings++
	_, _ = fmt.Fprintf(r.out, "[WARN] %s: %s\n", name, detail)
}

func (r *doctorReport) error(name, detail string) {
	r.errors++
	_, _ = fmt.Fprintf(r.out, "[FAIL] %s: %s\n", name, detail)
}

func (r *doctorReport) checkDirectory(name, path string) {
	info, err := os.Stat(path)
	if err != nil {
		r.error(name, err.Error())
		return
	}
	if !info.IsDir() {
		r.error(name, fmt.Sprintf("%s is not a directory", path))
		return
	}
	r.ok(name, path)
}

func (r *doctorReport) summary() {
	_, _ = fmt.Fprintf(r.out, "Summary: %d passed, %d warning(s), %d error(s)\n", r.passed, r.warnings, r.errors)
}

func (r *doctorReport) result() error {
	if r.errors == 0 {
		return nil
	}
	return fmt.Errorf("sdk doctor found %d error(s)", r.errors)
}
