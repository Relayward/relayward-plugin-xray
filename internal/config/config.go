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
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Relayward/relayward-sdk/contract"
)

const (
	VLESSRealityServiceID = "vless-reality"
	VLESSVisionFlow       = "xtls-rprx-vision"
)

var (
	serviceIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	domainPattern    = regexp.MustCompile(`^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+)$`)
)

type Configuration struct {
	XrayVersion    string       `json:"xray_version"`
	APIPort        uint16       `json:"api_port"`
	CredentialSeed string       `json:"credential_seed"`
	VLESSReality   VLESSReality `json:"vless_reality"`
}

type VLESSReality struct {
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
	XrayVersion  string               `json:"xray_version"`
	APIPort      uint16               `json:"api_port"`
	VLESSReality EditableVLESSReality `json:"vless_reality"`
}

type EditableVLESSReality struct {
	Enabled     bool   `json:"enabled"`
	DisplayName string `json:"display_name"`
	Listen      string `json:"listen"`
	Port        uint16 `json:"port"`
	PublicPort  uint16 `json:"public_port"`
	Target      string `json:"target"`
	ServerName  string `json:"server_name"`
	Fingerprint string `json:"fingerprint"`
}

func Editable(value Configuration) EditableConfiguration {
	serverName := ""
	if len(value.VLESSReality.ServerNames) > 0 {
		serverName = value.VLESSReality.ServerNames[0]
	}
	return EditableConfiguration{
		XrayVersion: value.XrayVersion,
		APIPort:     value.APIPort,
		VLESSReality: EditableVLESSReality{
			Enabled: value.VLESSReality.Enabled, DisplayName: value.VLESSReality.DisplayName,
			Listen: value.VLESSReality.Listen, Port: value.VLESSReality.Port,
			PublicPort: value.VLESSReality.PublicPort, Target: value.VLESSReality.Target,
			ServerName: serverName, Fingerprint: value.VLESSReality.Fingerprint,
		},
	}
}

func NewFromEditable(value EditableConfiguration) (Configuration, error) {
	configuration, err := NewConfiguration(value.XrayVersion, value.APIPort, value.VLESSReality.Port,
		value.VLESSReality.PublicPort, value.VLESSReality.Target, value.VLESSReality.ServerName)
	if err != nil {
		return Configuration{}, err
	}
	return MergeEditable(configuration, value)
}

func MergeEditable(configuration Configuration, value EditableConfiguration) (Configuration, error) {
	configuration.XrayVersion = value.XrayVersion
	configuration.APIPort = value.APIPort
	configuration.VLESSReality.Enabled = value.VLESSReality.Enabled
	configuration.VLESSReality.DisplayName = value.VLESSReality.DisplayName
	configuration.VLESSReality.Listen = value.VLESSReality.Listen
	configuration.VLESSReality.Port = value.VLESSReality.Port
	configuration.VLESSReality.PublicPort = value.VLESSReality.PublicPort
	configuration.VLESSReality.Target = value.VLESSReality.Target
	configuration.VLESSReality.ServerNames = []string{value.VLESSReality.ServerName}
	configuration.VLESSReality.Fingerprint = value.VLESSReality.Fingerprint
	if err := Validate(configuration); err != nil {
		return Configuration{}, err
	}
	return configuration, nil
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
	value.VLESSReality.ServerNames = append([]string(nil), value.VLESSReality.ServerNames...)
	value.VLESSReality.ShortIDs = append([]string(nil), value.VLESSReality.ShortIDs...)
	return value, nil
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
	service := value.VLESSReality
	if service.ServiceID != VLESSRealityServiceID || !serviceIDPattern.MatchString(service.ServiceID) {
		return fmt.Errorf("vless_reality.service_id: must be %q", VLESSRealityServiceID)
	}
	if err := validateDisplayName(service.DisplayName); err != nil {
		return fmt.Errorf("vless_reality.display_name: %w", err)
	}
	listen, err := netip.ParseAddr(service.Listen)
	if err != nil || listen.String() != service.Listen {
		return fmt.Errorf("vless_reality.listen: must be a canonical IP address")
	}
	if service.Port == 0 || service.PublicPort == 0 {
		return fmt.Errorf("vless_reality.port and public_port: must be between 1 and 65535")
	}
	if service.Port == value.APIPort && (listen.IsLoopback() || listen.IsUnspecified()) {
		return fmt.Errorf("vless_reality.port: conflicts with the local API port")
	}
	targetHost, targetPort, err := net.SplitHostPort(service.Target)
	parsedTargetPort, portErr := strconv.ParseUint(targetPort, 10, 16)
	if err != nil || portErr != nil || parsedTargetPort == 0 || !validServerName(targetHost) {
		return fmt.Errorf("vless_reality.target: must be a domain and port")
	}
	if len(service.ServerNames) == 0 || len(service.ServerNames) > 16 {
		return fmt.Errorf("vless_reality.server_names: must contain 1 to 16 values")
	}
	seenNames := make(map[string]struct{}, len(service.ServerNames))
	for index, name := range service.ServerNames {
		if !validServerName(name) {
			return fmt.Errorf("vless_reality.server_names[%d]: invalid server name", index)
		}
		if _, exists := seenNames[name]; exists {
			return fmt.Errorf("vless_reality.server_names[%d]: duplicate server name", index)
		}
		seenNames[name] = struct{}{}
	}
	if _, err := RealityPublicKey(service.PrivateKey); err != nil {
		return fmt.Errorf("vless_reality.private_key: %w", err)
	}
	if len(service.ShortIDs) == 0 || len(service.ShortIDs) > 16 {
		return fmt.Errorf("vless_reality.short_ids: must contain 1 to 16 values")
	}
	seenShortIDs := make(map[string]struct{}, len(service.ShortIDs))
	for index, shortID := range service.ShortIDs {
		decoded, err := hex.DecodeString(shortID)
		if err != nil || len(decoded) < 1 || len(decoded) > 8 {
			return fmt.Errorf("vless_reality.short_ids[%d]: must contain 2 to 16 lowercase hexadecimal characters", index)
		}
		if shortID != strings.ToLower(shortID) {
			return fmt.Errorf("vless_reality.short_ids[%d]: must use lowercase hexadecimal", index)
		}
		if _, exists := seenShortIDs[shortID]; exists {
			return fmt.Errorf("vless_reality.short_ids[%d]: duplicate short ID", index)
		}
		seenShortIDs[shortID] = struct{}{}
	}
	if service.Flow != VLESSVisionFlow {
		return fmt.Errorf("vless_reality.flow: must be %q", VLESSVisionFlow)
	}
	switch service.Fingerprint {
	case "chrome", "firefox", "safari", "ios", "android", "edge", "random", "randomized":
	default:
		return fmt.Errorf("vless_reality.fingerprint: unsupported fingerprint")
	}
	return nil
}

func NewConfiguration(xrayVersion string, apiPort, port, publicPort uint16, target, serverName string) (Configuration, error) {
	seed := make([]byte, 32)
	privateKey := make([]byte, 32)
	shortID := make([]byte, 8)
	for _, value := range [][]byte{seed, privateKey, shortID} {
		if _, err := rand.Read(value); err != nil {
			return Configuration{}, fmt.Errorf("generate configuration secret: %w", err)
		}
	}
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64
	value := Configuration{
		XrayVersion:    xrayVersion,
		APIPort:        apiPort,
		CredentialSeed: base64.RawURLEncoding.EncodeToString(seed),
		VLESSReality: VLESSReality{
			Enabled: true, ServiceID: VLESSRealityServiceID, DisplayName: "VLESS Reality",
			Listen: "0.0.0.0", Port: port, PublicPort: publicPort, Target: target,
			ServerNames: []string{serverName}, PrivateKey: base64.RawURLEncoding.EncodeToString(privateKey),
			ShortIDs: []string{hex.EncodeToString(shortID)}, Flow: VLESSVisionFlow, Fingerprint: "chrome",
		},
	}
	if err := Validate(value); err != nil {
		return Configuration{}, err
	}
	return value, nil
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

func (value Configuration) XrayJSON() ([]byte, error) {
	if err := Validate(value); err != nil {
		return nil, err
	}
	service := value.VLESSReality
	inbounds := []any{map[string]any{
		"tag": "relayward-api", "listen": "127.0.0.1", "port": value.APIPort,
		"protocol": "dokodemo-door", "settings": map[string]any{"address": "127.0.0.1"},
	}}
	if service.Enabled {
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
