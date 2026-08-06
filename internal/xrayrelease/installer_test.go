package xrayrelease

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type fakeSource struct {
	asset   Asset
	archive []byte
	resolve int
}

func (source *fakeSource) Resolve(context.Context, string) (Asset, error) {
	source.resolve++
	return source.asset, nil
}

func (source *fakeSource) Download(_ context.Context, _ Asset, destination io.Writer) error {
	_, err := destination.Write(source.archive)
	return err
}

func TestInstallerEnsuresAndReusesVerifiedVersion(t *testing.T) {
	t.Parallel()
	archive := testArchive(t, map[string][]byte{
		"xray": []byte("binary"), "geoip.dat": []byte("geoip"), "README.md": []byte("ignored"),
	})
	digest := sha256.Sum256(archive)
	source := &fakeSource{archive: archive, asset: Asset{
		Version: "26.3.27", URL: "unused", Size: int64(len(archive)), SHA256: hex.EncodeToString(digest[:]),
	}}
	installer := NewInstaller(t.TempDir(), source)
	installation, err := installer.Ensure(context.Background(), "26.3.27")
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if _, err := os.Stat(installation.Binary); err != nil {
		t.Fatalf("stat binary: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installation.AssetDir, "README.md")); !os.IsNotExist(err) {
		t.Fatalf("README should not be extracted, error = %v", err)
	}
	if _, err := installer.Ensure(context.Background(), "26.3.27"); err != nil {
		t.Fatalf("second Ensure() error = %v", err)
	}
	if source.resolve != 1 {
		t.Fatalf("Resolve() calls = %d, want 1", source.resolve)
	}
}

func TestInstallerRejectsUnsafeArchive(t *testing.T) {
	t.Parallel()
	archive := testArchive(t, map[string][]byte{"xray": []byte("binary"), "../escape": []byte("bad")})
	digest := sha256.Sum256(archive)
	source := &fakeSource{archive: archive, asset: Asset{
		Version: "26.3.27", Size: int64(len(archive)), SHA256: hex.EncodeToString(digest[:]),
	}}
	if _, err := NewInstaller(t.TempDir(), source).Ensure(context.Background(), "26.3.27"); err == nil {
		t.Fatal("Ensure() unexpectedly succeeded")
	}
}

func TestInstallerDetectsInstalledBinaryModification(t *testing.T) {
	t.Parallel()
	archive := testArchive(t, map[string][]byte{"xray": []byte("binary")})
	digest := sha256.Sum256(archive)
	source := &fakeSource{archive: archive, asset: Asset{
		Version: "26.3.27", Size: int64(len(archive)), SHA256: hex.EncodeToString(digest[:]),
	}}
	installer := NewInstaller(t.TempDir(), source)
	installation, err := installer.Ensure(context.Background(), "26.3.27")
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if err := os.Chmod(installation.AssetDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(installation.Binary, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installation.Binary, []byte("tampered"), 0o500); err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Ensure(context.Background(), "26.3.27"); err == nil {
		t.Fatal("Ensure() unexpectedly accepted modified binary")
	}
}

func testArchive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var raw bytes.Buffer
	writer := zip.NewWriter(&raw)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return raw.Bytes()
}
