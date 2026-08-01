package app

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
)

func printEnvironment(stateStore *store.Store, out io.Writer) error {
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	pathEntries := make([]string, 0, len(model.Tools()))
	for _, tool := range model.Tools() {
		if state.Defaults[tool] == "" {
			continue
		}
		currentPath := stateStore.CurrentPath(tool)
		for _, variable := range environmentVariables(tool, currentPath) {
			fmt.Fprintf(out, "export %s=%q\n", variable.name, variable.value)
		}
		pathEntries = append(pathEntries, filepath.Join(currentPath, "bin"))
	}
	if len(pathEntries) > 0 {
		fmt.Fprintf(out, "export PATH=%q:$PATH\n", strings.Join(pathEntries, ":"))
	}
	return nil
}

type environmentVariable struct {
	name  string
	value string
}

func environmentVariables(tool model.Tool, currentPath string) []environmentVariable {
	switch tool {
	case model.Java:
		return []environmentVariable{{name: "JAVA_HOME", value: currentPath}}
	case model.Maven:
		return []environmentVariable{{name: "MAVEN_HOME", value: currentPath}, {name: "M2_HOME", value: currentPath}}
	case model.Go:
		return []environmentVariable{{name: "GOROOT", value: currentPath}}
	case model.NodeJS:
		return []environmentVariable{{name: "NODE_HOME", value: currentPath}}
	default:
		return nil
	}
}
