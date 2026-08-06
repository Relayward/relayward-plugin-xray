package integration

import (
	"context"
	"net"
	"os"
	"strconv"
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
	apiPort := freePort(t)
	servicePort := freePort(t)
	configuration, err := config.NewConfiguration("26.3.27", apiPort, servicePort, servicePort,
		"www.microsoft.com:443", "www.microsoft.com")
	if err != nil {
		t.Fatal(err)
	}
	configuration.VLESSReality.Listen = "127.0.0.1"
	raw, err := config.Encode(configuration)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := agentv1.PluginConfigurationDigest(raw)
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
	authorizationID := "10000000-0000-4000-8000-000000000001"
	if err := runtime.ApplyServiceState(ctx, 1, 1, authorizationID, config.VLESSRealityServiceID, true); err != nil {
		t.Fatalf("ApplyServiceState() error = %v", err)
	}
	counters, err := runtime.CollectTraffic(ctx)
	if err != nil || len(counters) != 1 || counters[0].AuthorizationID != authorizationID {
		t.Fatalf("CollectTraffic() = %+v, %v", counters, err)
	}
	closeContext, closeCancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer closeCancel()
	if err := runtime.Close(closeContext); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func freePort(t *testing.T) uint16 {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, rawPort, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.ParseUint(rawPort, 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	return uint16(port)
}
