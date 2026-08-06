package center

import (
	"context"
	"sync"

	centerpluginv1 "github.com/Relayward/relayward-sdk/centerplugin/v1"
	"github.com/Relayward/relayward-sdk/contract"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Relayward/relayward-plugin-xray/internal/pluginmeta"
)

type Server struct {
	centerpluginv1.UnimplementedCenterPluginServer
	version string
	mu      sync.Mutex
	active  bool
}

func New(version string) *Server {
	return &Server{version: version}
}

func (server *Server) GetInfo(context.Context, *centerpluginv1.GetInfoRequest) (*centerpluginv1.GetInfoResponse, error) {
	return &centerpluginv1.GetInfoResponse{
		ApiVersion: contract.CenterPluginAPIVersion,
		PluginId:   pluginmeta.ID,
		Version:    server.version,
	}, nil
}

func (server *Server) Activate(_ context.Context, request *centerpluginv1.ActivateRequest) (*centerpluginv1.Activated, error) {
	if err := centerpluginv1.ValidateActivateRequest(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid activation request")
	}
	if len(request.Permissions) != 0 {
		return nil, status.Error(codes.InvalidArgument, "Xray plugin does not request center permissions")
	}
	server.mu.Lock()
	server.active = true
	server.mu.Unlock()
	return &centerpluginv1.Activated{Permissions: []string{}}, nil
}

func (server *Server) GetStatus(context.Context, *centerpluginv1.GetStatusRequest) (*centerpluginv1.GetStatusResponse, error) {
	server.mu.Lock()
	active := server.active
	server.mu.Unlock()
	health := centerpluginv1.Health_HEALTH_STARTING
	if active {
		health = centerpluginv1.Health_HEALTH_HEALTHY
	}
	return &centerpluginv1.GetStatusResponse{Health: health}, nil
}
