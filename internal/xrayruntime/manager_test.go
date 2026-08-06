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

type fakeRuntimeAPI struct{}

func (*fakeRuntimeAPI) addUser(context.Context, string, runtimeCredential) error { return nil }
func (*fakeRuntimeAPI) removeUser(context.Context, string, string) error         { return nil }
func (*fakeRuntimeAPI) close()                                                   {}
func (*fakeRuntimeAPI) queryStats(context.Context) ([]trafficStat, error) {
	return []trafficStat{
		{email: "relayward:10000000-0000-4000-8000-000000000001:vless-reality", direction: "uplink", value: 12},
		{email: "relayward:10000000-0000-4000-8000-000000000001:vless-reality", direction: "downlink", value: 34},
	}, nil
}

type failingTrafficRuntimeAPI struct {
	removed bool
}

func (*failingTrafficRuntimeAPI) addUser(context.Context, string, runtimeCredential) error {
	return nil
}
func (api *failingTrafficRuntimeAPI) removeUser(context.Context, string, string) error {
	api.removed = true
	return nil
}
func (*failingTrafficRuntimeAPI) close() {}
func (*failingTrafficRuntimeAPI) queryStats(context.Context) ([]trafficStat, error) {
	return nil, errors.New("traffic unavailable")
}

func (installer fakeInstaller) Ensure(context.Context, string) (xrayrelease.Installation, error) {
	return installer.installation, nil
}

func TestManagerAppliesAndStopsConfiguration(t *testing.T) {
	t.Parallel()
	manager := testManager(t)
	configuration := testConfigurationValue(t, "0.0.0.0")
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
	configuration := testConfigurationValue(t, "127.0.0.3")
	if err := manager.Validate(context.Background(), configuration); !errors.Is(err, ErrConfigurationRejected) {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestManagerRestoresPreviousProcessAfterCandidateStartupFailure(t *testing.T) {
	t.Parallel()
	manager := testManager(t)
	first := testConfigurationValue(t, "0.0.0.0")
	if err := manager.Apply(context.Background(), 1, digestA, first); err != nil {
		t.Fatalf("first Apply() error = %v", err)
	}
	second := testConfigurationValue(t, "127.0.0.2")
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
	configuration := testConfigurationValue(t, "127.0.0.4")
	if err := manager.Apply(context.Background(), 1, digestA, configuration); err == nil {
		t.Fatal("Apply() unexpectedly succeeded")
	}
	if status := manager.GetStatus(); status.Generation != 0 || status.Healthy {
		t.Fatalf("GetStatus() = %+v", status)
	}
}

func TestManagerControlsUsersAndCollectsTraffic(t *testing.T) {
	t.Parallel()
	manager := testManager(t)
	configuration := testConfigurationValue(t, "0.0.0.0")
	if err := manager.Apply(context.Background(), 1, digestA, configuration); err != nil {
		t.Fatal(err)
	}
	authorizationID := "10000000-0000-4000-8000-000000000001"
	if err := manager.ApplyServiceState(context.Background(), 1, 1, authorizationID, config.VLESSRealityServiceID, true); err != nil {
		t.Fatalf("ApplyServiceState(enable) error = %v", err)
	}
	counters, err := manager.CollectTraffic(context.Background())
	if err != nil {
		t.Fatalf("CollectTraffic() error = %v", err)
	}
	if len(counters) != 1 || counters[0].AuthorizationID != authorizationID || counters[0].UploadBytes != 12 || counters[0].DownloadBytes != 34 || counters[0].CounterEpoch == "" {
		t.Fatalf("CollectTraffic() = %+v", counters)
	}
	if err := manager.ApplyServiceState(context.Background(), 1, 2, authorizationID, config.VLESSRealityServiceID, false); err != nil {
		t.Fatalf("ApplyServiceState(disable) error = %v", err)
	}
	if err := manager.ApplyServiceState(context.Background(), 1, 1, authorizationID, config.VLESSRealityServiceID, true); !errors.Is(err, ErrServiceStateConflict) {
		t.Fatalf("ApplyServiceState(stale) error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestManagerKeepsUserEnabledWhenFinalTrafficCollectionFails(t *testing.T) {
	t.Parallel()
	manager := testManager(t)
	configuration := testConfigurationValue(t, "0.0.0.0")
	if err := manager.Apply(context.Background(), 1, digestA, configuration); err != nil {
		t.Fatal(err)
	}
	authorizationID := "10000000-0000-4000-8000-000000000001"
	if err := manager.ApplyServiceState(context.Background(), 1, 1, authorizationID, config.VLESSRealityServiceID, true); err != nil {
		t.Fatal(err)
	}
	failingAPI := &failingTrafficRuntimeAPI{}
	manager.process.api = failingAPI
	if err := manager.ApplyServiceState(context.Background(), 1, 2, authorizationID, config.VLESSRealityServiceID, false); err == nil {
		t.Fatal("ApplyServiceState(disable) unexpectedly succeeded")
	}
	state := manager.services[serviceKey(authorizationID, config.VLESSRealityServiceID)]
	if state == nil || !state.enabled || failingAPI.removed {
		t.Fatalf("service state = %+v, removed = %v", state, failingAPI.removed)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

func testConfigurationValue(t *testing.T, listen string) config.Configuration {
	t.Helper()
	value, err := config.NewConfiguration("26.3.27", 10085, 443, 443, "www.microsoft.com:443", "www.microsoft.com")
	if err != nil {
		t.Fatal(err)
	}
	value.VLESSReality.Listen = listen
	if err := config.Validate(value); err != nil {
		t.Fatal(err)
	}
	return value
}

func testManager(t *testing.T) *Manager {
	t.Helper()
	directory := t.TempDir()
	binary := filepath.Join(directory, "fake-xray")
	script := `#!/bin/sh
if [ "$1" = "api" ]; then
  case "$2" in
    statsquery)
      printf '%s\n' '{"stat":[{"name":"user>>>relayward:10000000-0000-4000-8000-000000000001:vless-reality>>>traffic>>>uplink","value":"12"},{"name":"user>>>relayward:10000000-0000-4000-8000-000000000001:vless-reality>>>traffic>>>downlink","value":"34"}]}'
      ;;
  esac
  exit 0
fi
config=""
test_mode=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    -test) test_mode=1 ;;
    -config) shift; config=$1 ;;
  esac
  shift
done
if grep -q '"listen":"127.0.0.3"' "$config"; then
  exit 1
fi
if [ "$test_mode" -eq 1 ]; then
  exit 0
fi
if [ "$test_mode" -eq 0 ] && grep -q '"listen":"127.0.0.2"' "$config"; then
  exit 1
fi
if [ "$test_mode" -eq 0 ] && grep -q '"listen":"127.0.0.4"' "$config"; then
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
	manager.connectAPI = func(context.Context, config.Configuration) (runtimeAPI, error) {
		return &fakeRuntimeAPI{}, nil
	}
	return manager
}

const (
	digestA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)
