// Package config owns the structured Xray configuration understood by the official Relayward plugin.
package config

import (
	"bytes"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/netip"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Relayward/relayward-sdk/contract"
)

const (
	ServiceTypeVLESSReality = "vless-reality"
	VLESSVisionFlow         = "xtls-rprx-vision"
	MaximumServices         = 64
)

var (
	serviceIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	domainPattern    = regexp.MustCompile(`^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+)$`)
)

type Configuration struct {
	XrayVersion    string    `json:"xray_version"`
	APIPort        uint16    `json:"api_port"`
	CredentialSeed string    `json:"credential_seed"`
	Services       []Service `json:"services"`
}

type Service struct {
	Type        string   `json:"type"`
	Enabled     bool     `json:"enabled"`
	ServiceID   string   `json:"service_id"`
	DisplayName string   `json:"display_name"`
	Listen      string   `json:"listen"`
	Port        uint16   `json:"port"`
	PublicPort  uint16   `json:"public_port"`
	Target      string   `json:"target"`
	ServerNames []string `json:"server_names"`
	PrivateKey  string   `json:"private_key"`
	ShortIDs    []string `json:"short_ids"`
	Flow        string   `json:"flow"`
	Fingerprint string   `json:"fingerprint"`
}

type EditableConfiguration struct {
	XrayVersion string            `json:"xray_version"`
	APIPort     uint16            `json:"api_port"`
	Services    []EditableService `json:"services"`
}

type EditableService struct {
	Type        string `json:"type"`
	Enabled     bool   `json:"enabled"`
	ServiceID   string `json:"service_id"`
	DisplayName string `json:"display_name"`
	Listen      string `json:"listen"`
	Port        uint16 `json:"port"`
	PublicPort  uint16 `json:"public_port"`
	Target      string `json:"target"`
	ServerName  string `json:"server_name"`
	Fingerprint string `json:"fingerprint"`
}

func Editable(value Configuration) EditableConfiguration {
	services := make([]EditableService, len(value.Services))
	for index, service := range value.Services {
		serverName := ""
		if len(service.ServerNames) > 0 {
			serverName = service.ServerNames[0]
		}
		services[index] = EditableService{
			Type: service.Type, Enabled: service.Enabled, ServiceID: service.ServiceID,
			DisplayName: service.DisplayName, Listen: service.Listen, Port: service.Port,
			PublicPort: service.PublicPort, Target: service.Target, ServerName: serverName,
			Fingerprint: service.Fingerprint,
		}
	}
	return EditableConfiguration{XrayVersion: value.XrayVersion, APIPort: value.APIPort, Services: services}
}

func NewConfiguration(xrayVersion string, apiPort uint16, services []EditableService) (Configuration, error) {
	return NewFromEditable(EditableConfiguration{XrayVersion: xrayVersion, APIPort: apiPort, Services: services})
}

func NewFromEditable(value EditableConfiguration) (Configuration, error) {
	seed, err := randomKey()
	if err != nil {
		return Configuration{}, err
	}
	configuration := Configuration{CredentialSeed: seed}
	return MergeEditable(configuration, value)
}

func MergeEditable(configuration Configuration, value EditableConfiguration) (Configuration, error) {
	existing := make(map[string]Service, len(configuration.Services))
	for _, service := range configuration.Services {
		existing[service.ServiceID] = service
	}
	services := make([]Service, len(value.Services))
	for index, editable := range value.Services {
		service, exists := existing[editable.ServiceID]
		if !exists || service.Type != editable.Type {
			var err error
			service, err = newServiceSecrets()
			if err != nil {
				return Configuration{}, err
			}
		}
		service.Type = editable.Type
		service.Enabled = editable.Enabled
		service.ServiceID = editable.ServiceID
		service.DisplayName = editable.DisplayName
		service.Listen = editable.Listen
		service.Port = editable.Port
		service.PublicPort = editable.PublicPort
		service.Target = editable.Target
		service.ServerNames = []string{editable.ServerName}
		service.Flow = VLESSVisionFlow
		service.Fingerprint = editable.Fingerprint
		services[index] = service
	}
	configuration.XrayVersion = value.XrayVersion
	configuration.APIPort = value.APIPort
	sort.Slice(services, func(i, j int) bool { return services[i].ServiceID < services[j].ServiceID })
	configuration.Services = services
	if err := Validate(configuration); err != nil {
		return Configuration{}, err
	}
	return clone(configuration), nil
}

func Decode(raw []byte) (Configuration, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var value Configuration
	if err := decoder.Decode(&value); err != nil {
		return Configuration{}, fmt.Errorf("decode configuration: %w", err)
	}
	if err := requireEOF(decoder); err != nil {
		return Configuration{}, err
	}
	if err := Validate(value); err != nil {
		return Configuration{}, err
	}
	return clone(value), nil
}

func Validate(value Configuration) error {
	if err := contract.ValidateSemanticVersion(value.XrayVersion); err != nil {
		return fmt.Errorf("xray_version: %w", err)
	}
	if strings.ContainsAny(value.XrayVersion, "-+") {
		return fmt.Errorf("xray_version: pre-release and build versions are not supported")
	}
	if value.APIPort < 1024 {
		return fmt.Errorf("api_port: must be between 1024 and 65535")
	}
	if _, err := decodeKey("credential_seed", value.CredentialSeed); err != nil {
		return err
	}
	if len(value.Services) > MaximumServices {
		return fmt.Errorf("services: must contain at most %d services", MaximumServices)
	}
	seenIDs := make(map[string]struct{}, len(value.Services))
	for index, service := range value.Services {
		field := fmt.Sprintf("services[%d]", index)
		if service.Type != ServiceTypeVLESSReality {
			return fmt.Errorf("%s.type: unsupported service type", field)
		}
		if !serviceIDPattern.MatchString(service.ServiceID) {
			return fmt.Errorf("%s.service_id: must match %s", field, serviceIDPattern)
		}
		if _, exists := seenIDs[service.ServiceID]; exists {
			return fmt.Errorf("%s.service_id: duplicate service ID", field)
		}
		seenIDs[service.ServiceID] = struct{}{}
		if index > 0 && value.Services[index-1].ServiceID > service.ServiceID {
			return fmt.Errorf("%s.service_id: services must be sorted by service ID", field)
		}
		if err := validateService(value.APIPort, service, field); err != nil {
			return err
		}
		for previousIndex := 0; previousIndex < index; previousIndex++ {
			previous := value.Services[previousIndex]
			if service.Port == previous.Port && listenersOverlap(service.Listen, previous.Listen) {
				return fmt.Errorf("%s.port: conflicts with services[%d]", field, previousIndex)
			}
		}
	}
	return nil
}

func validateService(apiPort uint16, service Service, field string) error {
	if err := validateDisplayName(service.DisplayName); err != nil {
		return fmt.Errorf("%s.display_name: %w", field, err)
	}
	listen, err := netip.ParseAddr(service.Listen)
	if err != nil || listen.String() != service.Listen {
		return fmt.Errorf("%s.listen: must be a canonical IP address", field)
	}
	if service.Port == 0 || service.PublicPort == 0 {
		return fmt.Errorf("%s.port and public_port: must be between 1 and 65535", field)
	}
	if service.Port == apiPort && (listen.IsLoopback() || listen.IsUnspecified()) {
		return fmt.Errorf("%s.port: conflicts with the local API port", field)
	}
	targetHost, targetPort, err := net.SplitHostPort(service.Target)
	parsedTargetPort, portErr := strconv.ParseUint(targetPort, 10, 16)
	if err != nil || portErr != nil || parsedTargetPort == 0 || !validServerName(targetHost) {
		return fmt.Errorf("%s.target: must be a domain and port", field)
	}
	if len(service.ServerNames) == 0 || len(service.ServerNames) > 16 {
		return fmt.Errorf("%s.server_names: must contain 1 to 16 values", field)
	}
	seenNames := make(map[string]struct{}, len(service.ServerNames))
	for index, name := range service.ServerNames {
		if !validServerName(name) {
			return fmt.Errorf("%s.server_names[%d]: invalid server name", field, index)
		}
		if _, exists := seenNames[name]; exists {
			return fmt.Errorf("%s.server_names[%d]: duplicate server name", field, index)
		}
		seenNames[name] = struct{}{}
	}
	if _, err := RealityPublicKey(service.PrivateKey); err != nil {
		return fmt.Errorf("%s.private_key: %w", field, err)
	}
	if len(service.ShortIDs) == 0 || len(service.ShortIDs) > 16 {
		return fmt.Errorf("%s.short_ids: must contain 1 to 16 values", field)
	}
	seenShortIDs := make(map[string]struct{}, len(service.ShortIDs))
	for index, shortID := range service.ShortIDs {
		decoded, err := hex.DecodeString(shortID)
		if err != nil || len(decoded) < 1 || len(decoded) > 8 {
			return fmt.Errorf("%s.short_ids[%d]: must contain 2 to 16 lowercase hexadecimal characters", field, index)
		}
		if shortID != strings.ToLower(shortID) {
			return fmt.Errorf("%s.short_ids[%d]: must use lowercase hexadecimal", field, index)
		}
		if _, exists := seenShortIDs[shortID]; exists {
			return fmt.Errorf("%s.short_ids[%d]: duplicate short ID", field, index)
		}
		seenShortIDs[shortID] = struct{}{}
	}
	if service.Flow != VLESSVisionFlow {
		return fmt.Errorf("%s.flow: must be %q", field, VLESSVisionFlow)
	}
	switch service.Fingerprint {
	case "chrome", "firefox", "safari", "ios", "android", "edge", "random", "randomized":
	default:
		return fmt.Errorf("%s.fingerprint: unsupported fingerprint", field)
	}
	return nil
}

func Encode(value Configuration) ([]byte, error) {
	if err := Validate(value); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode configuration: %w", err)
	}
	return raw, nil
}

func (value Configuration) FindService(serviceID string) (Service, bool) {
	for _, service := range value.Services {
		if service.ServiceID == serviceID {
			return service, true
		}
	}
	return Service{}, false
}

func (value Configuration) XrayJSON() ([]byte, error) {
	if err := Validate(value); err != nil {
		return nil, err
	}
	inbounds := []any{map[string]any{
		"tag": "relayward-api", "listen": "127.0.0.1", "port": value.APIPort,
		"protocol": "dokodemo-door", "settings": map[string]any{"address": "127.0.0.1"},
	}}
	for _, service := range value.Services {
		if !service.Enabled {
			continue
		}
		inbounds = append(inbounds, map[string]any{
			"tag": service.ServiceID, "listen": service.Listen, "port": service.Port, "protocol": "vless",
			"settings": map[string]any{"clients": []any{}, "decryption": "none"},
			"streamSettings": map[string]any{
				"network": "tcp", "security": "reality",
				"realitySettings": map[string]any{
					"show": false, "target": service.Target, "xver": 0, "serverNames": service.ServerNames,
					"privateKey": service.PrivateKey, "shortIds": service.ShortIDs,
				},
			},
		})
	}
	result := map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"api": map[string]any{"tag": "relayward-api", "services": []string{
			"HandlerService", "RoutingService", "StatsService",
		}},
		"inbounds": inbounds,
		"outbounds": []any{
			map[string]any{"tag": "direct", "protocol": "freedom", "settings": map[string]any{}},
			map[string]any{"tag": "blocked", "protocol": "blackhole", "settings": map[string]any{}},
		},
		"policy": map[string]any{"levels": map[string]any{"0": map[string]any{
			"statsUserUplink": true, "statsUserDownlink": true, "statsUserOnline": true,
		}}},
		"routing": map[string]any{"rules": []any{map[string]any{
			"type": "field", "inboundTag": []string{"relayward-api"}, "outboundTag": "relayward-api",
		}}},
		"stats": map[string]any{},
	}
	return json.Marshal(result)
}

func DeriveCredential(seed, authorizationID, serviceID string) (string, error) {
	key, err := decodeKey("credential_seed", seed)
	if err != nil {
		return "", err
	}
	if authorizationID == "" || serviceID == "" || strings.ContainsRune(authorizationID, 0) || strings.ContainsRune(serviceID, 0) {
		return "", fmt.Errorf("authorization and service IDs are required")
	}
	hash := hmac.New(sha256.New, key)
	_, _ = hash.Write([]byte(authorizationID))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(serviceID))
	value := hash.Sum(nil)[:16]
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}

func RealityPublicKey(privateKey string) (string, error) {
	raw, err := decodeKey("private key", privateKey)
	if err != nil {
		return "", err
	}
	key, err := ecdh.X25519().NewPrivateKey(raw)
	if err != nil {
		return "", fmt.Errorf("invalid X25519 private key")
	}
	return base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()), nil
}

func UserEmail(authorizationID, serviceID string) string {
	return "relayward:" + authorizationID + ":" + serviceID
}

func newServiceSecrets() (Service, error) {
	privateKey := make([]byte, 32)
	shortID := make([]byte, 8)
	for _, value := range [][]byte{privateKey, shortID} {
		if _, err := rand.Read(value); err != nil {
			return Service{}, fmt.Errorf("generate configuration secret: %w", err)
		}
	}
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64
	return Service{
		PrivateKey: base64.RawURLEncoding.EncodeToString(privateKey),
		ShortIDs:   []string{hex.EncodeToString(shortID)},
	}, nil
}

func randomKey() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate configuration secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func clone(value Configuration) Configuration {
	value.Services = append([]Service(nil), value.Services...)
	for index := range value.Services {
		value.Services[index].ServerNames = append([]string(nil), value.Services[index].ServerNames...)
		value.Services[index].ShortIDs = append([]string(nil), value.Services[index].ShortIDs...)
	}
	return value
}

func listenersOverlap(first, second string) bool {
	firstAddress, firstErr := netip.ParseAddr(first)
	secondAddress, secondErr := netip.ParseAddr(second)
	if firstErr != nil || secondErr != nil {
		return false
	}
	return firstAddress == secondAddress || firstAddress.IsUnspecified() || secondAddress.IsUnspecified()
}

func decodeKey(field, value string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(raw) != 32 || base64.RawURLEncoding.EncodeToString(raw) != value {
		return nil, fmt.Errorf("%s: must be 32 bytes of unpadded base64url", field)
	}
	return raw, nil
}

func validServerName(value string) bool {
	if _, err := netip.ParseAddr(value); err == nil {
		return false
	}
	return len(value) <= 253 && domainPattern.MatchString(value)
}

func validateDisplayName(value string) error {
	if value == "" || value != strings.TrimSpace(value) || !utf8.ValidString(value) || utf8.RuneCountInString(value) > 100 {
		return fmt.Errorf("must contain 1 to 100 trimmed characters")
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return fmt.Errorf("must not contain control characters")
		}
	}
	return nil
}

func requireEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("configuration contains a trailing JSON value")
		}
		return fmt.Errorf("decode trailing configuration: %w", err)
	}
	return nil
}
