package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const repository = "Denxuan/sdk"

type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Updater struct {
	HTTP       *http.Client
	Executable func() (string, error)
}

func New() *Updater {
	return &Updater{
		HTTP:       &http.Client{Timeout: 5 * time.Minute},
		Executable: os.Executable,
	}
}

func (u *Updater) Update(ctx context.Context, requestedVersion string) (string, error) {
	release, err := u.release(ctx, requestedVersion)
	if err != nil {
		return "", err
	}
	asset, err := selectAsset(release.Assets, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", fmt.Errorf("select release %s asset: %w", release.TagName, err)
	}
	executable, err := u.Executable()
	if err != nil {
		return "", fmt.Errorf("find current executable: %w", err)
	}
	if err := u.replaceFromAsset(ctx, asset, executable); err != nil {
		return "", err
	}
	return release.TagName, nil
}

func (u *Updater) release(ctx context.Context, requestedVersion string) (Release, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repository)
	if requestedVersion != "" {
		endpoint = fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repository, normalizeTag(requestedVersion))
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Release{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := u.HTTP.Do(request)
	if err != nil {
		return Release{}, fmt.Errorf("request release metadata: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("request release metadata: GitHub returned %s", response.Status)
	}
	var release Release
	if err := json.NewDecoder(io.LimitReader(response.Body, 10<<20)).Decode(&release); err != nil {
		return Release{}, fmt.Errorf("decode release metadata: %w", err)
	}
	return release, nil
}

func (u *Updater) replaceFromAsset(ctx context.Context, asset Asset, executable string) error {
	directory := filepath.Dir(executable)
	temporary, err := os.MkdirTemp(directory, ".sdk-update-")
	if err != nil {
		return fmt.Errorf("create update directory: %w", err)
	}
	defer os.RemoveAll(temporary)
	archivePath := filepath.Join(temporary, asset.Name)
	if err := u.download(ctx, asset.DownloadURL, archivePath); err != nil {
		return err
	}
	candidate := filepath.Join(temporary, filepath.Base(executable))
	if err := extractExecutable(archivePath, candidate); err != nil {
		return err
	}
	if err := os.Chmod(candidate, 0755); err != nil {
		return fmt.Errorf("mark downloaded sdk executable: %w", err)
	}
	if err := replaceExecutable(candidate, executable); err != nil {
		return fmt.Errorf("replace sdk executable (Homebrew installs should use `brew upgrade sdk`): %w", err)
	}
	return nil
}

func (u *Updater) download(ctx context.Context, url, target string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := u.HTTP.Do(request)
	if err != nil {
		return fmt.Errorf("download update asset: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download update asset: server returned %s", response.Status)
	}
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, io.LimitReader(response.Body, 2<<30))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func selectAsset(assets []Asset, operatingSystem, architecture string) (Asset, error) {
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if !strings.Contains(name, strings.ToLower(operatingSystem)) || !strings.Contains(name, strings.ToLower(architecture)) {
			continue
		}
		if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip") {
			return asset, nil
		}
	}
	return Asset{}, fmt.Errorf("no archive for %s/%s", operatingSystem, architecture)
}

func extractExecutable(archivePath, target string) error {
	if strings.HasSuffix(strings.ToLower(archivePath), ".zip") {
		return extractZIPExecutable(archivePath, target)
	}
	if strings.HasSuffix(strings.ToLower(archivePath), ".tar.gz") {
		return extractTarGzipExecutable(archivePath, target)
	}
	return fmt.Errorf("unsupported update archive %s", archivePath)
}

func extractTarGzipExecutable(archivePath, target string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeReg && isSDKExecutable(header.Name) {
			return copyExecutable(target, reader)
		}
	}
	return errors.New("update archive does not contain sdk executable")
}

func extractZIPExecutable(archivePath, target string) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()
	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() || !isSDKExecutable(entry.Name) {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return err
		}
		err = copyExecutable(target, reader)
		reader.Close()
		return err
	}
	return errors.New("update archive does not contain sdk executable")
}

func isSDKExecutable(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return name == "sdk" || name == "sdk.exe"
}

func copyExecutable(target string, source io.Reader) error {
	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, source)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func replaceExecutable(source, target string) error {
	backup := target + ".previous"
	if err := os.Remove(backup); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(target, backup); err != nil {
		return err
	}
	if err := os.Rename(source, target); err != nil {
		_ = os.Rename(backup, target)
		return err
	}
	return os.Remove(backup)
}

func normalizeTag(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}
