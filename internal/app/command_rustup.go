package app

import (
	"context"
	"fmt"
	"io"

	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/rustup"
	"github.com/Denxuan/sdk/internal/store"
)

func rustupCommand(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: sdk rustup <args...>")
	}
	return rustup.Run(ctx, out, args...)
}

func uninstallRustup(ctx context.Context, stateStore *store.Store, out io.Writer) error {
	if err := rustup.SelfUninstall(ctx, out); err != nil {
		return err
	}
	state, err := stateStore.Load()
	if err != nil {
		return err
	}
	delete(state.Defaults, model.Rust)
	delete(state.Installed, model.Rust)
	if err := stateStore.RemoveCurrent(model.Rust); err != nil {
		return err
	}
	if err := stateStore.Save(state); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(out, "uninstalled rustup")
	return nil
}
