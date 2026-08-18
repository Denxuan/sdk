package app

import (
	"fmt"
	"io"

	"github.com/Denxuan/sdk/internal/installer"
	"github.com/Denxuan/sdk/internal/model"
)

type downloadProgress struct {
	out     io.Writer
	tool    model.Tool
	version string
	active  bool
}

func newDownloadProgress(out io.Writer, tool model.Tool, version string) *downloadProgress {
	return &downloadProgress{out: out, tool: tool, version: version}
}

func (p *downloadProgress) Update(progress installer.Progress) {
	p.active = true
	if progress.Total > 0 {
		percent := progress.Downloaded * 100 / progress.Total
		_, _ = fmt.Fprintf(p.out, "\rDownloading %s %s: %3d%% (%s / %s)", p.tool, p.version, percent, formatBytes(progress.Downloaded), formatBytes(progress.Total))
		return
	}
	_, _ = fmt.Fprintf(p.out, "\rDownloading %s %s: %s", p.tool, p.version, formatBytes(progress.Downloaded))
}

func (p *downloadProgress) Finish() {
	if p.active {
		_, _ = fmt.Fprintln(p.out)
	}
}

func (p *downloadProgress) Retry(attempt, total int, err error) {
	if p.active {
		_, _ = fmt.Fprintln(p.out)
	}
	_, _ = fmt.Fprintf(p.out, "Download interrupted; retrying %s %s (%d/%d): %v\n", p.tool, p.version, attempt, total, err)
}

func (p *downloadProgress) Verify(checksum installer.Checksum) {
	if p.active {
		_, _ = fmt.Fprintln(p.out)
	}
	_, _ = fmt.Fprintf(p.out, "Verifying %s checksum for %s %s...\n", checksum.Algorithm, p.tool, p.version)
}

func formatBytes(bytes int64) string {
	const kilobyte = 1024
	const megabyte = 1024 * 1024
	if bytes < kilobyte {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < megabyte {
		return fmt.Sprintf("%d KiB", bytes/kilobyte)
	}
	return fmt.Sprintf("%.1f MiB", float64(bytes)/megabyte)
}
