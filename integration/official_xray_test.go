package integration

import (
	"context"
	"os"
	"testing"
	"time"

	agentv1 "github.com/Relayward/relayward-sdk/agent/v1"

	"github.com/Relayward/relayward-plugin-xray/internal/config"
	"github.com/Relayward/relayward-plugin-xray/internal/xrayrelease"
	"github.com/Relayward/relayward-plugin-xray/internal/xrayruntime"
)

func TestOfficialXrayLifecycle(t *testing.T) {
	if os.Getenv("RELAYWARD_XRAY_INTEGRATION") != "1" {
		t.Skip("set RELAYWARD_XRAY_INTEGRATION=1 to download and run official Xray")
	}
	const raw = `{"xray_version":"26.3.27","xray_config":{"log":{"loglevel":"none"}}}`
	configuration, err := config.Decode([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	digest, err := agentv1.PluginConfigurationDigest([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	installer := xrayrelease.NewInstaller(directory, xrayrelease.NewClient())
	runtime := xrayruntime.NewManager(directory, installer)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := runtime.Validate(ctx, configuration); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := runtime.Apply(ctx, 1, digest, configuration); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	status := runtime.GetStatus()
	if !status.Healthy || status.Generation != 1 || status.ConfigurationSHA256 != digest {
		t.Fatalf("GetStatus() = %+v", status)
	}
	closeContext, closeCancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer closeCancel()
	if err := runtime.Close(closeContext); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
