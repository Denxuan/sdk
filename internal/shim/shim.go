package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Denxuan/sdk/internal/model"
)

var commands = map[model.Tool][]string{
	model.Java:   {"java", "javac", "jar"},
	model.Maven:  {"mvn"},
	model.Go:     {"go", "gofmt"},
	model.NodeJS: {"node", "npm", "npx"},
}

func ToolFor(command string) (model.Tool, bool) {
	for tool, names := range commands {
		for _, name := range names {
			if name == command {
				return tool, true
			}
		}
	}
	return "", false
}

// Ensure writes small launchers that forward tool commands to sdk exec.
func Ensure(home, sdkBinary string) error {
	directory := filepath.Join(home, "shims")
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	for _, names := range commands {
		for _, name := range names {
			if runtime.GOOS == "windows" {
				contents := fmt.Sprintf("@echo off\r\n\"%s\" exec %s %%*\r\n", sdkBinary, name)
				if err := os.WriteFile(filepath.Join(directory, name+".cmd"), []byte(contents), 0755); err != nil {
					return err
				}
				continue
			}
			contents := fmt.Sprintf("#!/bin/sh\nexec %q exec %q \"$@\"\n", sdkBinary, name)
			if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0755); err != nil {
				return err
			}
		}
	}
	return nil
}
