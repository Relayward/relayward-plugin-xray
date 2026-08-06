package node

import (
	"context"
	"errors"

	"github.com/Relayward/relayward-sdk/contract"
	nodepluginv1 "github.com/Relayward/relayward-sdk/nodeplugin/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Relayward/relayward-plugin-xray/internal/config"
	"github.com/Relayward/relayward-plugin-xray/internal/pluginmeta"
	"github.com/Relayward/relayward-plugin-xray/internal/xrayruntime"
)

type Runtime interface {
	Validate(context.Context, config.Configuration) error
	Apply(context.Context, uint64, string, config.Configuration) error
	GetStatus() xrayruntime.Status
}

type Server struct {
	nodepluginv1.UnimplementedNodePluginServer
	version string
	runtime Runtime
}

func New(version string, runtime Runtime) *Server {
	return &Server{version: version, runtime: runtime}
}

func (server *Server) GetInfo(context.Context, *nodepluginv1.GetInfoRequest) (*nodepluginv1.GetInfoResponse, error) {
	return &nodepluginv1.GetInfoResponse{
		ApiVersion:   contract.NodePluginAPIVersion,
		PluginId:     pluginmeta.ID,
		Version:      server.version,
		Capabilities: []string{},
	}, nil
}

func (server *Server) ValidateConfiguration(ctx context.Context, request *nodepluginv1.ConfigurationRequest) (*nodepluginv1.ConfigurationValidated, error) {
	if err := nodepluginv1.ValidateConfigurationRequest(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid configuration envelope")
	}
	configuration, err := config.Decode(request.Json)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid Xray plugin configuration")
	}
	if err := server.runtime.Validate(ctx, configuration); err != nil {
		if errors.Is(err, xrayruntime.ErrConfigurationRejected) {
			return nil, status.Error(codes.InvalidArgument, "Xray rejected the configuration")
		}
		return nil, status.Error(codes.Unavailable, "Xray runtime preparation failed")
	}
	return &nodepluginv1.ConfigurationValidated{Generation: request.Generation, Sha256: request.Sha256}, nil
}

func (server *Server) ApplyConfiguration(ctx context.Context, request *nodepluginv1.ConfigurationRequest) (*nodepluginv1.ConfigurationApplied, error) {
	if err := nodepluginv1.ValidateConfigurationRequest(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid configuration envelope")
	}
	configuration, err := config.Decode(request.Json)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid Xray plugin configuration")
	}
	if err := server.runtime.Apply(ctx, request.Generation, request.Sha256, configuration); err != nil {
		if errors.Is(err, xrayruntime.ErrConfigurationRejected) {
			return nil, status.Error(codes.InvalidArgument, "Xray rejected the configuration")
		}
		return nil, status.Error(codes.Internal, "Xray runtime switch failed")
	}
	return &nodepluginv1.ConfigurationApplied{Generation: request.Generation, Sha256: request.Sha256}, nil
}

func (server *Server) GetStatus(context.Context, *nodepluginv1.GetStatusRequest) (*nodepluginv1.GetStatusResponse, error) {
	runtimeStatus := server.runtime.GetStatus()
	health := nodepluginv1.Health_HEALTH_STARTING
	if runtimeStatus.Generation != 0 {
		health = nodepluginv1.Health_HEALTH_UNHEALTHY
		if runtimeStatus.Healthy {
			health = nodepluginv1.Health_HEALTH_HEALTHY
		}
	}
	return &nodepluginv1.GetStatusResponse{
		Generation:          runtimeStatus.Generation,
		ConfigurationSha256: runtimeStatus.ConfigurationSHA256,
		Health:              health,
		Message:             runtimeStatus.Message,
	}, nil
}
