package node

import (
	"context"
	"testing"

	agentv1 "github.com/Relayward/relayward-sdk/agent/v1"
	nodepluginv1 "github.com/Relayward/relayward-sdk/nodeplugin/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Relayward/relayward-plugin-xray/internal/config"
	"github.com/Relayward/relayward-plugin-xray/internal/xrayruntime"
)

type fakeRuntime struct {
	status        xrayruntime.Status
	applied       config.Configuration
	serviceState  bool
	serviceID     string
	authorization string
}

func (*fakeRuntime) Validate(context.Context, config.Configuration) error { return nil }
func (runtime *fakeRuntime) Apply(_ context.Context, generation uint64, digest string, configuration config.Configuration) error {
	runtime.applied = configuration
	runtime.status = xrayruntime.Status{Generation: generation, ConfigurationSHA256: digest, Healthy: true}
	return nil
}
func (runtime *fakeRuntime) GetStatus() xrayruntime.Status { return runtime.status }
func (runtime *fakeRuntime) ApplyServiceState(_ context.Context, _, _ uint64, authorizationID, serviceID string, enabled bool) error {
	runtime.authorization = authorizationID
	runtime.serviceID = serviceID
	runtime.serviceState = enabled
	return nil
}
func (*fakeRuntime) CollectTraffic(context.Context) ([]xrayruntime.TrafficCounter, error) {
	return []xrayruntime.TrafficCounter{{
		AuthorizationID: "10000000-0000-4000-8000-000000000001",
		ServiceID:       config.VLESSRealityServiceID, CounterEpoch: "epoch-1", UploadBytes: 12, DownloadBytes: 34,
	}}, nil
}

func TestServerAppliesConfiguration(t *testing.T) {
	t.Parallel()
	runtime := &fakeRuntime{}
	server := New("0.1.0", runtime)
	raw := testConfigurationJSON(t)
	digest, err := agentv1.PluginConfigurationDigest(raw)
	if err != nil {
		t.Fatal(err)
	}
	request := &nodepluginv1.ConfigurationRequest{Generation: 3, Sha256: digest, Json: raw}
	validated, err := server.ValidateConfiguration(context.Background(), request)
	if err != nil {
		t.Fatalf("ValidateConfiguration() error = %v", err)
	}
	if err := nodepluginv1.ValidateConfigurationValidated(request, validated); err != nil {
		t.Fatal(err)
	}
	applied, err := server.ApplyConfiguration(context.Background(), request)
	if err != nil {
		t.Fatalf("ApplyConfiguration() error = %v", err)
	}
	if err := nodepluginv1.ValidateConfigurationApplied(request, applied); err != nil {
		t.Fatal(err)
	}
	result, err := server.GetStatus(context.Background(), &nodepluginv1.GetStatusRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if err := nodepluginv1.ValidateStatusResponse(result); err != nil {
		t.Fatal(err)
	}
	if result.Health != nodepluginv1.Health_HEALTH_HEALTHY || runtime.applied.XrayVersion != "26.3.27" {
		t.Fatalf("GetStatus() = %+v, applied = %+v", result, runtime.applied)
	}
}

func TestServerControlsServiceAndReturnsTraffic(t *testing.T) {
	t.Parallel()
	runtime := &fakeRuntime{}
	server := New("0.1.0", runtime)
	info, err := server.GetInfo(context.Background(), &nodepluginv1.GetInfoRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if err := nodepluginv1.ValidateInfoResponse(info, info.PluginId, info.Version); err != nil {
		t.Fatal(err)
	}
	authorizationID := "10000000-0000-4000-8000-000000000001"
	state := &nodepluginv1.SetServiceStateRequest{
		PolicyGeneration: 1, StateRevision: 2, AuthorizationId: authorizationID,
		ServiceId: config.VLESSRealityServiceID, Enabled: true,
		Reason: nodepluginv1.ServiceStateReason_SERVICE_STATE_REASON_ACTIVE,
	}
	applied, err := server.SetServiceState(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	if err := nodepluginv1.ValidateSetServiceStateResponse(state, applied); err != nil {
		t.Fatal(err)
	}
	if !runtime.serviceState || runtime.authorization != authorizationID || runtime.serviceID != config.VLESSRealityServiceID {
		t.Fatalf("runtime state = %+v", runtime)
	}
	request := &nodepluginv1.CollectTelemetryRequest{MaximumEvents: 10}
	telemetry, err := server.CollectTelemetry(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if err := nodepluginv1.ValidateCollectTelemetryResponse(request, telemetry); err != nil {
		t.Fatal(err)
	}
	if len(telemetry.Counters) != 1 || telemetry.Counters[0].UploadBytes != 12 {
		t.Fatalf("telemetry = %+v", telemetry)
	}
}

func TestServerRejectsUnknownConfigurationField(t *testing.T) {
	t.Parallel()
	valid := testConfigurationJSON(t)
	raw := append(append([]byte(nil), valid[:len(valid)-1]...), []byte(`,"secret":"value"}`)...)
	digest, err := agentv1.PluginConfigurationDigest(raw)
	if err != nil {
		t.Fatal(err)
	}
	_, err = New("0.1.0", &fakeRuntime{}).ValidateConfiguration(context.Background(), &nodepluginv1.ConfigurationRequest{
		Generation: 1, Sha256: digest, Json: raw,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ValidateConfiguration() code = %v, want InvalidArgument", status.Code(err))
	}
}

func testConfigurationJSON(t *testing.T) []byte {
	t.Helper()
	value, err := config.NewConfiguration("26.3.27", 10085, 443, 443, "www.microsoft.com:443", "www.microsoft.com")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := config.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
