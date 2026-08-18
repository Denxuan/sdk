package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Denxuan/sdk/internal/selfupdate"
)

func selfUpdate(ctx context.Context, args []string, out io.Writer) error {
	if len(args) > 1 {
		return errors.New("usage: sdk selfupdate [version]")
	}
	requestedVersion := ""
	if len(args) == 1 {
		requestedVersion = args[0]
	}
	_, _ = fmt.Fprintln(out, "Checking sdk releases...")
	version, err := selfupdate.New().Update(ctx, requestedVersion)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "Updated sdk to %s\n", version)
	return nil
}
