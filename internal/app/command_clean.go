package app

import (
	"errors"
	"fmt"
	"io"

	"github.com/Denxuan/sdk/internal/installer"
	"github.com/Denxuan/sdk/internal/store"
)

func cleanDownloads(stateStore *store.Store, args []string, out io.Writer) error {
	if len(args) != 0 {
		return errors.New("usage: sdk clean")
	}
	removed, err := installer.CleanPartialDownloads(stateStore.ToolsDir())
	if err != nil {
		return fmt.Errorf("clean incomplete downloads: %w", err)
	}
	_, _ = fmt.Fprintf(out, "removed %d incomplete download(s)\n", removed)
	return nil
}
