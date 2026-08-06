package xrayruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Relayward/relayward-plugin-xray/internal/config"
	"github.com/Relayward/relayward-plugin-xray/internal/xrayrelease"
)

type fakeInstaller struct {
	installation xrayrelease.Installation
}

func (installer fakeInstaller) Ensure(context.Context, string) (xrayrelease.Installation, error) {
	return installer.installation, nil
}

func TestManagerAppliesAndStopsConfiguration(t *testing.T) {
	t.Parallel()
	manager := testManager(t)
	configuration := config.Configuration{XrayVersion: "26.3.27", XrayConfig: []byte(`{"log":{}}`)}
	if err := manager.Validate(context.Background(), configuration); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := manager.Apply(context.Background(), 7, digestA, configuration); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	status := manager.GetStatus()
	if !status.Healthy || status.Generation != 7 || status.ConfigurationSHA256 != digestA {
		t.Fatalf("GetStatus() = %+v", status)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestManagerRejectsInvalidXrayConfiguration(t *testing.T) {
	t.Parallel()
	manager := testManager(t)
	configuration := config.Configuration{XrayVersion: "26.3.27", XrayConfig: []byte(`{"reject":true}`)}
	if err := manager.Validate(context.Background(), configuration); !errors.Is(err, ErrConfigurationRejected) {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestManagerRestoresPreviousProcessAfterCandidateStartupFailure(t *testing.T) {
	t.Parallel()
	manager := testManager(t)
	first := config.Configuration{XrayVersion: "26.3.27", XrayConfig: []byte(`{"name":"first"}`)}
	if err := manager.Apply(context.Background(), 1, digestA, first); err != nil {
		t.Fatalf("first Apply() error = %v", err)
	}
	second := config.Configuration{XrayVersion: "26.3.27", XrayConfig: []byte(`{"start_fail":true}`)}
	if err := manager.Apply(context.Background(), 2, digestB, second); err == nil {
		t.Fatal("second Apply() unexpectedly succeeded")
	}
	status := manager.GetStatus()
	if !status.Healthy || status.Generation != 1 || status.ConfigurationSHA256 != digestA {
		t.Fatalf("GetStatus() after rollback = %+v", status)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestManagerRejectsProcessThatExitsCleanlyDuringStartup(t *testing.T) {
	t.Parallel()
	manager := testManager(t)
	configuration := config.Configuration{XrayVersion: "26.3.27", XrayConfig: []byte(`{"exit_cleanly":true}`)}
	if err := manager.Apply(context.Background(), 1, digestA, configuration); err == nil {
		t.Fatal("Apply() unexpectedly succeeded")
	}
	if status := manager.GetStatus(); status.Generation != 0 || status.Healthy {
		t.Fatalf("GetStatus() = %+v", status)
	}
}

func testManager(t *testing.T) *Manager {
	t.Helper()
	directory := t.TempDir()
	binary := filepath.Join(directory, "fake-xray")
	script := `#!/bin/sh
config=""
test_mode=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    -test) test_mode=1 ;;
    -config) shift; config=$1 ;;
  esac
  shift
done
if grep -q '"reject":true' "$config"; then
  exit 1
fi
if [ "$test_mode" -eq 1 ]; then
  exit 0
fi
if [ "$test_mode" -eq 0 ] && grep -q '"start_fail":true' "$config"; then
  exit 1
fi
if [ "$test_mode" -eq 0 ] && grep -q '"exit_cleanly":true' "$config"; then
  exit 0
fi
trap 'exit 0' TERM INT
while :; do sleep 1; done
`
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(directory, fakeInstaller{installation: xrayrelease.Installation{
		Version: "26.3.27", Binary: binary, AssetDir: directory,
	}})
	manager.startupGrace = 50 * time.Millisecond
	return manager
}

const (
	digestA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)
