package installer

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Progress struct {
	Downloaded int64
	Total      int64
}

type ProgressReporter func(Progress)
type RetryReporter func(attempt, total int, err error)
type VerificationReporter func(Checksum)

type Checksum struct {
	Algorithm string
	Value     string
}

const defaultDownloadAttempts = 3

type Installer struct {
	HTTP        *http.Client
	Progress    ProgressReporter
	Retry       RetryReporter
	Verify      VerificationReporter
	MaxAttempts int
	RetryDelay  func(attempt int) time.Duration
	Sleep       func(time.Duration)
}

func New() *Installer {
	return &Installer{
		HTTP:        &http.Client{Timeout: 5 * time.Minute},
		MaxAttempts: defaultDownloadAttempts,
		RetryDelay: func(attempt int) time.Duration {
			return time.Second * time.Duration(1<<(attempt-1))
		},
		Sleep: time.Sleep,
	}
}

func (i *Installer) WithProgress(reporter ProgressReporter) *Installer {
	i.Progress = reporter
	return i
}

func (i *Installer) WithRetry(reporter RetryReporter) *Installer {
	i.Retry = reporter
	return i
}

func (i *Installer) WithVerification(reporter VerificationReporter) *Installer {
	i.Verify = reporter
	return i
}

// Install downloads an archive, extracts it into destination, and returns the
// final directory. The archive's single top-level folder is removed.
func (i *Installer) Install(ctx context.Context, url, destination string, checksum Checksum) error {
	if checksum.Value == "" {
		return errors.New("installation requires an upstream checksum")
	}
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("installation directory already exists: %s", destination)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	temp, err := os.MkdirTemp(filepath.Dir(destination), ".install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	archive := filepath.Join(temp, "package")
	if err := i.download(ctx, url, archive); err != nil {
		return err
	}
	if i.Verify != nil {
		i.Verify(checksum)
	}
	if err := verifyChecksum(archive, checksum); err != nil {
		return err
	}
	extracted := filepath.Join(temp, "extracted")
	if err := extract(archive, extracted); err != nil {
		return err
	}
	root, err := archiveRoot(extracted)
	if err != nil {
		return err
	}
	if err := os.Rename(root, destination); err != nil {
		return fmt.Errorf("finalize installation: %w", err)
	}
	return nil
}

func verifyChecksum(path string, checksum Checksum) error {
	expected, err := hex.DecodeString(strings.TrimSpace(checksum.Value))
	if err != nil {
		return fmt.Errorf("decode expected %s checksum: %w", checksum.Algorithm, err)
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open downloaded archive for checksum: %w", err)
	}
	defer file.Close()

	var actual []byte
	switch strings.ToLower(checksum.Algorithm) {
	case "sha256":
		hash := sha256.New()
		_, err = io.Copy(hash, file)
		actual = hash.Sum(nil)
	case "sha512":
		hash := sha512.New()
		_, err = io.Copy(hash, file)
		actual = hash.Sum(nil)
	default:
		return fmt.Errorf("unsupported checksum algorithm %q", checksum.Algorithm)
	}
	if err != nil {
		return fmt.Errorf("calculate %s checksum: %w", checksum.Algorithm, err)
	}
	if !bytes.Equal(actual, expected) {
		return fmt.Errorf("%s checksum mismatch for downloaded archive", checksum.Algorithm)
	}
	return nil
}

func (i *Installer) download(ctx context.Context, url, target string) error {
	attempts := i.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		err := i.downloadOnce(ctx, url, target)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt == attempts || !isRetryableDownloadError(err) {
			break
		}
		if i.Retry != nil {
			i.Retry(attempt+1, attempts, err)
		}
		if err := waitForRetry(ctx, i.retryDelay(attempt), i.sleep()); err != nil {
			return err
		}
	}
	return fmt.Errorf("download failed after %d attempt(s): %w", attempts, lastErr)
}

func (i *Installer) downloadOnce(ctx context.Context, url, target string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := i.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &downloadStatusError{URL: url, StatusCode: resp.StatusCode, Status: resp.Status}
	}
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	writer := &progressWriter{writer: file, total: resp.ContentLength, report: i.Progress}
	_, copyErr := io.Copy(writer, io.LimitReader(resp.Body, 2<<30))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

type downloadStatusError struct {
	URL        string
	StatusCode int
	Status     string
}

func (e *downloadStatusError) Error() string {
	return fmt.Sprintf("download %s: server returned %s", e.URL, e.Status)
}

func isRetryableDownloadError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var statusError *downloadStatusError
	if errors.As(err, &statusError) {
		return statusError.StatusCode == http.StatusRequestTimeout || statusError.StatusCode == http.StatusTooManyRequests || statusError.StatusCode >= 500
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var networkError net.Error
	return errors.As(err, &networkError)
}

func (i *Installer) retryDelay(attempt int) time.Duration {
	if i.RetryDelay == nil {
		return 0
	}
	return i.RetryDelay(attempt)
}

func (i *Installer) sleep() func(time.Duration) {
	if i.Sleep == nil {
		return time.Sleep
	}
	return i.Sleep
}

func waitForRetry(ctx context.Context, delay time.Duration, sleep func(time.Duration)) error {
	if delay <= 0 {
		return nil
	}
	completed := make(chan struct{})
	go func() { sleep(delay); close(completed) }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-completed:
		return nil
	}
}

type progressWriter struct {
	writer     io.Writer
	total      int64
	downloaded int64
	report     ProgressReporter
	lastReport time.Time
}

func (w *progressWriter) Write(data []byte) (int, error) {
	written, err := w.writer.Write(data)
	w.downloaded += int64(written)
	w.reportProgress()
	return written, err
}

func (w *progressWriter) reportProgress() {
	if w.report == nil {
		return
	}
	now := time.Now()
	finished := w.total > 0 && w.downloaded >= w.total
	if !w.lastReport.IsZero() && !finished && now.Sub(w.lastReport) < 100*time.Millisecond {
		return
	}
	w.lastReport = now
	w.report(Progress{Downloaded: w.downloaded, Total: w.total})
}

func extract(archive, destination string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	header := make([]byte, 4)
	if _, err := io.ReadFull(file, header); err != nil {
		return fmt.Errorf("read archive: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if string(header[:2]) == "PK" {
		return extractZIP(file, destination)
	}
	if header[0] == 0x1f && header[1] == 0x8b {
		gz, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gz.Close()
		return extractTAR(tar.NewReader(gz), destination)
	}
	return fmt.Errorf("unsupported archive format")
}

func extractZIP(reader *os.File, destination string) error {
	info, err := reader.Stat()
	if err != nil {
		return err
	}
	archive, err := zip.NewReader(reader, info.Size())
	if err != nil {
		return err
	}
	for _, entry := range archive.File {
		target, err := safePath(destination, entry.Name)
		if err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, entry.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		source, err := entry.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, entry.Mode())
		if err == nil {
			_, err = io.Copy(output, source)
			closeErr := output.Close()
			if err == nil {
				err = closeErr
			}
		}
		source.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTAR(reader *tar.Reader, destination string) error {
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safePath(destination, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(output, reader)
			closeErr := output.Close()
			if err == nil {
				err = closeErr
			}
			if err != nil {
				return err
			}
		}
	}
}

func safePath(root, name string) (string, error) {
	target := filepath.Join(root, filepath.Clean(name))
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive contains unsafe path %q", name)
	}
	return target, nil
}

func archiveRoot(directory string) (string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return "", fmt.Errorf("archive must contain one top-level directory")
	}
	return filepath.Join(directory, entries[0].Name()), nil
}
