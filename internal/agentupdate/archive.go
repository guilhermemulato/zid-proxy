package agentupdate

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ExtractVersionAndBinary(archivePath, outDir, expectedBinaryName string) (version string, extractedBinaryPath string, err error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var foundVersion string
	var foundBinary string

	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", err
		}

		name := filepath.Clean(h.Name)
		base := filepath.Base(name)

		switch {
		case base == "VERSION":
			if h.FileInfo().Mode().IsRegular() {
				data, err := io.ReadAll(io.LimitReader(tr, 64))
				if err != nil {
					return "", "", err
				}
				foundVersion = strings.TrimSpace(string(data))
			}
		case base == expectedBinaryName:
			if h.FileInfo().Mode().IsRegular() {
				outPath := filepath.Join(outDir, expectedBinaryName)
				if err := writeFileFromTar(outPath, tr, 0o755); err != nil {
					return "", "", err
				}
				foundBinary = outPath
			}
		}
	}

	if foundVersion == "" {
		return "", "", fmt.Errorf("VERSION not found in archive")
	}
	if foundBinary == "" {
		return "", "", fmt.Errorf("binary %q not found in archive", expectedBinaryName)
	}

	return foundVersion, foundBinary, nil
}

func writeFileFromTar(outPath string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	tmp := outPath + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, r)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, outPath)
}
