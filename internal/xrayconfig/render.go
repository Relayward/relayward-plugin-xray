package xrayconfig

import (
	"encoding/json"
	"fmt"

	"github.com/Relayward/relayward-plugin-xray/internal/config"
)

func SupportsServiceType(serviceType string) bool {
	return serviceType == config.ServiceTypeVLESSReality
}

func Render(value config.Configuration) ([]byte, error) {
	if err := config.Validate(value); err != nil {
		return nil, err
	}
	routingRules, err := CompileRoutingRules(value, nil)
	if err != nil {
		return nil, err
	}
	sniffing := NeedsSniffing(value)
	inbounds := []any{map[string]any{
		"tag": "relayward-api", "listen": "127.0.0.1", "port": value.APIPort,
		"protocol": "dokodemo-door", "settings": map[string]any{"address": "127.0.0.1"},
	}}
	for _, service := range value.Services {
		if !service.Enabled {
			continue
		}
		inbound, err := renderService(service, sniffing)
		if err != nil {
			return nil, err
		}
		inbounds = append(inbounds, inbound)
	}
	directSettings := map[string]any{}
	routing := map[string]any{"rules": renderRoutingRules(routingRules)}
	if value.DNS.Enabled {
		directSettings["domainStrategy"] = xrayDNSQueryStrategy(value.DNS.QueryStrategy)
		routing["domainStrategy"] = "IPIfNonMatch"
	}
	result := map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"api": map[string]any{"tag": "relayward-api", "services": []string{
			"HandlerService", "RoutingService", "StatsService",
		}},
		"inbounds": inbounds,
		"outbounds": []any{
			map[string]any{"tag": "direct", "protocol": "freedom", "settings": directSettings},
			map[string]any{"tag": "blocked", "protocol": "blackhole", "settings": map[string]any{}},
		},
		"policy": map[string]any{"levels": map[string]any{"0": map[string]any{
			"statsUserUplink": true, "statsUserDownlink": true, "statsUserOnline": true,
		}}},
		"routing": routing,
		"stats":   map[string]any{},
	}
	if value.DNS.Enabled {
		result["dns"] = renderDNS(value.DNS)
	}
	return json.Marshal(result)
}

func renderService(service config.Service, sniffing bool) (any, error) {
	switch service.Type {
	case config.ServiceTypeVLESSReality:
		return renderVLESSReality(service, sniffing), nil
	default:
		return nil, fmt.Errorf("unsupported Xray service type %q", service.Type)
	}
}

func renderVLESSReality(service config.Service, sniffing bool) any {
	reality := service.VLESSReality
	inbound := map[string]any{
		"tag": service.ServiceID, "listen": service.Listen, "port": service.Port, "protocol": "vless",
		"settings": map[string]any{"clients": []any{}, "decryption": "none"},
		"streamSettings": map[string]any{
			"network": "tcp", "security": "reality",
			"realitySettings": map[string]any{
				"show": false, "target": reality.Target, "xver": 0, "serverNames": reality.ServerNames,
				"privateKey": reality.PrivateKey, "shortIds": reality.ShortIDs,
			},
		},
	}
	if sniffing {
		inbound["sniffing"] = map[string]any{
			"enabled": true, "destOverride": []string{"http", "tls", "quic"}, "routeOnly": true,
		}
	}
	return inbound
}
