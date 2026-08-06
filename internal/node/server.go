package node

import (
	"context"
	"errors"
	"time"

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
	ApplyServiceState(context.Context, uint64, uint64, string, string, bool) error
	CollectTraffic(context.Context) ([]xrayruntime.TrafficCounter, error)
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
		Capabilities: []string{nodepluginv1.CapabilityServiceControl, nodepluginv1.CapabilityTrafficCounters},
	}, nil
}

func (server *Server) CollectTelemetry(ctx context.Context, request *nodepluginv1.CollectTelemetryRequest) (*nodepluginv1.CollectTelemetryResponse, error) {
	if err := nodepluginv1.ValidateCollectTelemetryRequest(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid telemetry request")
	}
	values, err := server.runtime.CollectTraffic(ctx)
	if err != nil {
		if errors.Is(err, xrayruntime.ErrRuntimeUnavailable) {
			return nil, status.Error(codes.FailedPrecondition, "Xray runtime is unavailable")
		}
		return nil, status.Error(codes.Unavailable, "Xray traffic collection failed")
	}
	response := &nodepluginv1.CollectTelemetryResponse{
		ObservedAtUnixNano: time.Now().UTC().UnixNano(),
		Counters:           make([]*nodepluginv1.TrafficCounter, len(values)),
		NextSequence:       request.AfterSequence,
	}
	for index, value := range values {
		response.Counters[index] = &nodepluginv1.TrafficCounter{
			AuthorizationId: value.AuthorizationID,
			ServiceId:       value.ServiceID,
			CounterEpoch:    value.CounterEpoch,
			UploadBytes:     value.UploadBytes,
			DownloadBytes:   value.DownloadBytes,
		}
	}
	if err := nodepluginv1.ValidateCollectTelemetryResponse(request, response); err != nil {
		return nil, status.Error(codes.Internal, "Xray returned invalid traffic counters")
	}
	return response, nil
}

func (server *Server) SetServiceState(ctx context.Context, request *nodepluginv1.SetServiceStateRequest) (*nodepluginv1.SetServiceStateResponse, error) {
	if err := nodepluginv1.ValidateSetServiceStateRequest(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid service state request")
	}
	if err := server.runtime.ApplyServiceState(ctx, request.PolicyGeneration, request.StateRevision,
		request.AuthorizationId, request.ServiceId, request.Enabled); err != nil {
		switch {
		case errors.Is(err, xrayruntime.ErrUnsupportedService):
			return nil, status.Error(codes.InvalidArgument, "unsupported Xray service")
		case errors.Is(err, xrayruntime.ErrRuntimeUnavailable), errors.Is(err, xrayruntime.ErrServiceDisabled):
			return nil, status.Error(codes.FailedPrecondition, "Xray service is unavailable")
		case errors.Is(err, xrayruntime.ErrServiceStateConflict):
			return nil, status.Error(codes.Aborted, "Xray service state revision changed")
		default:
			return nil, status.Error(codes.Unavailable, "Xray service state update failed")
		}
	}
	return &nodepluginv1.SetServiceStateResponse{
		PolicyGeneration: request.PolicyGeneration,
		StateRevision:    request.StateRevision,
		AuthorizationId:  request.AuthorizationId,
		ServiceId:        request.ServiceId,
		Enabled:          request.Enabled,
		Reason:           request.Reason,
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
