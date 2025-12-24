package agentupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractVersionAndBinary(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "bundle.tar.gz")

	wantVersion := "1.2.3"
	wantBin := "zid-agent-linux-gui"

	if err := writeTestTarGz(archivePath, map[string][]byte{
		"zid-agent-linux-gui/VERSION":             []byte(wantVersion + "\n"),
		"zid-agent-linux-gui/zid-agent-linux-gui": []byte("fakebin"),
	}); err != nil {
		t.Fatalf("writeTestTarGz: %v", err)
	}

	gotVersion, gotBinPath, err := ExtractVersionAndBinary(archivePath, dir, wantBin)
	if err != nil {
		t.Fatalf("ExtractVersionAndBinary: %v", err)
	}
	if gotVersion != wantVersion {
		t.Fatalf("version mismatch: got=%q want=%q", gotVersion, wantVersion)
	}
	if filepath.Base(gotBinPath) != wantBin {
		t.Fatalf("binary mismatch: got=%q want base=%q", gotBinPath, wantBin)
	}
	data, err := os.ReadFile(gotBinPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "fakebin" {
		t.Fatalf("binary content mismatch: got=%q", string(data))
	}
}

func writeTestTarGz(path string, files map[string][]byte) error {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, data := range files {
		h := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(data)),
		}
		if err := tw.WriteHeader(h); err != nil {
			_ = tw.Close()
			_ = gzw.Close()
			return err
		}
		if _, err := tw.Write(data); err != nil {
			_ = tw.Close()
			_ = gzw.Close()
			return err
		}
	}

	if err := tw.Close(); err != nil {
		_ = gzw.Close()
		return err
	}
	if err := gzw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o600)
}
