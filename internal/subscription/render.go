package subscription

import (
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strconv"

	centerpluginv1 "github.com/Relayward/relayward-sdk/centerplugin/v1"

	"github.com/Relayward/relayward-plugin-xray/internal/config"
)

func SupportsServiceType(serviceType string) bool {
	return serviceType == config.ServiceTypeVLESSReality
}

func SupportedFormats(serviceType string) []string {
	if !SupportsServiceType(serviceType) {
		return nil
	}
	return []string{"base64", "mihomo", "sing-box"}
}

func Render(configuration config.Configuration, request *centerpluginv1.RenderSubscriptionRequest) (*centerpluginv1.RenderSubscriptionResponse, error) {
	if err := centerpluginv1.ValidateRenderSubscriptionRequest(request); err != nil {
		return nil, err
	}
	if err := config.Validate(configuration); err != nil {
		return nil, errors.New("stored Xray configuration is invalid")
	}
	response := &centerpluginv1.RenderSubscriptionResponse{
		Services: make([]*centerpluginv1.SubscriptionServiceContribution, len(request.Services)),
	}
	for index, binding := range request.Services {
		service, exists := configuration.FindService(binding.ServiceId)
		if !exists {
			return nil, errors.New("subscription requests an unsupported Xray service")
		}
		if !service.Enabled {
			return nil, errors.New("subscription requests a disabled Xray service")
		}
		contribution, err := renderService(configuration, service, binding, request.AuthorizationId)
		if err != nil {
			return nil, err
		}
		response.Services[index] = contribution
	}
	if err := centerpluginv1.ValidateRenderSubscriptionResponse(request, response); err != nil {
		return nil, err
	}
	return response, nil
}

func renderService(configuration config.Configuration, service config.Service,
	binding *centerpluginv1.SubscriptionServiceBinding, authorizationID string,
) (*centerpluginv1.SubscriptionServiceContribution, error) {
	switch service.Type {
	case config.ServiceTypeVLESSReality:
		return renderVLESSReality(configuration, service, binding, authorizationID)
	default:
		return nil, errors.New("subscription requests an unsupported Xray service type")
	}
}

func renderVLESSReality(configuration config.Configuration, service config.Service,
	binding *centerpluginv1.SubscriptionServiceBinding, authorizationID string,
) (*centerpluginv1.SubscriptionServiceContribution, error) {
	reality := service.VLESSReality
	publicKey, err := config.RealityPublicKey(reality.PrivateKey)
	if err != nil {
		return nil, err
	}
	serverName := reality.ServerNames[0]
	shortID := reality.ShortIDs[0]
	credential, err := config.DeriveCredential(configuration.CredentialSeed, authorizationID, binding.ServiceId)
	if err != nil {
		return nil, err
	}
	uri := vlessURI(service.PublicHost, service.PublicPort, credential, binding.DisplayName, reality.Flow,
		reality.Fingerprint, serverName, publicKey, shortID)
	mihomo, err := json.Marshal(map[string]any{
		"name": binding.DisplayName, "type": "vless", "server": service.PublicHost, "port": service.PublicPort,
		"uuid": credential, "network": "tcp", "tls": true, "udp": true, "flow": reality.Flow,
		"servername": serverName, "client-fingerprint": reality.Fingerprint,
		"reality-opts": map[string]any{"public-key": publicKey, "short-id": shortID},
	})
	if err != nil {
		return nil, err
	}
	singBox, err := json.Marshal(map[string]any{
		"type": "vless", "tag": binding.DisplayName, "server": service.PublicHost, "server_port": service.PublicPort,
		"uuid": credential, "flow": reality.Flow,
		"tls": map[string]any{
			"enabled": true, "server_name": serverName,
			"utls":    map[string]any{"enabled": true, "fingerprint": reality.Fingerprint},
			"reality": map[string]any{"enabled": true, "public_key": publicKey, "short_id": shortID},
		},
	})
	if err != nil {
		return nil, err
	}
	return &centerpluginv1.SubscriptionServiceContribution{
		ServiceId: binding.ServiceId, DisplayName: binding.DisplayName,
		Uris: []string{uri}, MihomoProxiesJson: [][]byte{mihomo}, SingBoxOutboundsJson: [][]byte{singBox},
	}, nil
}

func vlessURI(host string, port uint16, credential, displayName, flow, fingerprint, serverName, publicKey, shortID string) string {
	value := &url.URL{
		Scheme: "vless", User: url.User(credential), Host: net.JoinHostPort(host, strconv.Itoa(int(port))),
		Fragment: displayName,
	}
	query := url.Values{
		"encryption": {"none"}, "flow": {flow}, "fp": {fingerprint}, "pbk": {publicKey},
		"security": {"reality"}, "sid": {shortID}, "sni": {serverName}, "type": {"tcp"},
	}
	value.RawQuery = query.Encode()
	return value.String()
}
