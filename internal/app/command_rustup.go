package app

import (
	"context"
	"fmt"
	"io"

	"github.com/Denxuan/sdk/internal/rustup"
)

func rustupCommand(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: sdk rustup <args...>")
	}
	return rustup.Run(ctx, out, args...)
}
