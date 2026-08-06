package center

import (
	"context"
	"testing"

	centerpluginv1 "github.com/Relayward/relayward-sdk/centerplugin/v1"
	"github.com/Relayward/relayward-sdk/contract"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Relayward/relayward-plugin-xray/internal/pluginmeta"
)

func TestServerLifecycle(t *testing.T) {
	t.Parallel()
	server := New("0.1.0")
	info, err := server.GetInfo(context.Background(), &centerpluginv1.GetInfoRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if info.ApiVersion != contract.CenterPluginAPIVersion || info.PluginId != pluginmeta.ID || info.Version != "0.1.0" {
		t.Fatalf("GetInfo() = %+v", info)
	}
	before, err := server.GetStatus(context.Background(), &centerpluginv1.GetStatusRequest{})
	if err != nil || before.Health != centerpluginv1.Health_HEALTH_STARTING {
		t.Fatalf("GetStatus() before activation = %+v, %v", before, err)
	}
	activation := &centerpluginv1.ActivateRequest{Permissions: []string{}}
	activated, err := server.Activate(context.Background(), activation)
	if err != nil {
		t.Fatal(err)
	}
	if err := centerpluginv1.ValidateActivated(activation, activated); err != nil {
		t.Fatalf("ValidateActivated() error = %v", err)
	}
	after, err := server.GetStatus(context.Background(), &centerpluginv1.GetStatusRequest{})
	if err != nil || after.Health != centerpluginv1.Health_HEALTH_HEALTHY {
		t.Fatalf("GetStatus() after activation = %+v, %v", after, err)
	}
}

func TestServerRejectsPermissions(t *testing.T) {
	t.Parallel()
	_, err := New("0.1.0").Activate(context.Background(), &centerpluginv1.ActivateRequest{Permissions: []string{centerpluginv1.PermissionNodesRead}})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Activate() code = %v, want InvalidArgument", status.Code(err))
	}
}
