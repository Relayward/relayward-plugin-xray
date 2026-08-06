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
	status  xrayruntime.Status
	applied config.Configuration
}

func (*fakeRuntime) Validate(context.Context, config.Configuration) error { return nil }
func (runtime *fakeRuntime) Apply(_ context.Context, generation uint64, digest string, configuration config.Configuration) error {
	runtime.applied = configuration
	runtime.status = xrayruntime.Status{Generation: generation, ConfigurationSHA256: digest, Healthy: true}
	return nil
}
func (runtime *fakeRuntime) GetStatus() xrayruntime.Status { return runtime.status }

func TestServerAppliesConfiguration(t *testing.T) {
	t.Parallel()
	runtime := &fakeRuntime{}
	server := New("0.1.0", runtime)
	raw := []byte(`{"xray_version":"26.3.27","xray_config":{}}`)
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

func TestServerRejectsUnknownConfigurationField(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"xray_version":"26.3.27","xray_config":{},"secret":"value"}`)
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
