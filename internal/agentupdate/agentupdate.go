package agentupdate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type PreparedUpdate struct {
	TempDir     string
	ArchivePath string
	Version     string
	BinaryPath  string
}

func (pu PreparedUpdate) Cleanup() {
	if pu.TempDir == "" {
		return
	}
	_ = os.RemoveAll(pu.TempDir)
}

type Downloader struct {
	Client  *http.Client
	Timeout time.Duration
}

func (d Downloader) httpClient() *http.Client {
	if d.Client != nil {
		return d.Client
	}
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

func (d Downloader) DownloadToFile(ctx context.Context, url, outPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := d.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, err = io.Copy(f, resp.Body)
	return err
}

func PrepareFromURL(ctx context.Context, downloader Downloader, url, expectedBinaryName string) (PreparedUpdate, error) {
	tmpDir, err := os.MkdirTemp("", "zid-agent-update-*")
	if err != nil {
		return PreparedUpdate{}, err
	}

	archivePath := filepath.Join(tmpDir, "bundle.tar.gz")
	if err := downloader.DownloadToFile(ctx, url, archivePath); err != nil {
		_ = os.RemoveAll(tmpDir)
		return PreparedUpdate{}, err
	}

	version, binPath, err := ExtractVersionAndBinary(archivePath, tmpDir, expectedBinaryName)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return PreparedUpdate{}, err
	}

	return PreparedUpdate{
		TempDir:     tmpDir,
		ArchivePath: archivePath,
		Version:     version,
		BinaryPath:  binPath,
	}, nil
}
