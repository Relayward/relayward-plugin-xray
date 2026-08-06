package config

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
)

var domainPattern = regexp.MustCompile(`^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+)$`)

type VLESSReality struct {
	Target      string   `json:"target"`
	ServerNames []string `json:"server_names"`
	PrivateKey  string   `json:"private_key"`
	ShortIDs    []string `json:"short_ids"`
	Flow        string   `json:"flow"`
	Fingerprint string   `json:"fingerprint"`
}

type EditableVLESSReality struct {
	Target      string `json:"target"`
	ServerName  string `json:"server_name"`
	Fingerprint string `json:"fingerprint"`
}

type ServiceTypeCapabilities struct {
	XrayInbound         bool     `json:"xray_inbound"`
	ServiceControl      bool     `json:"service_control"`
	TrafficCounters     bool     `json:"traffic_counters"`
	RecentActivity      bool     `json:"recent_activity"`
	DynamicBlocking     bool     `json:"dynamic_blocking"`
	SubscriptionFormats []string `json:"subscription_formats"`
}

type ServiceTypeDefinition struct {
	ID           string                  `json:"id"`
	DisplayName  string                  `json:"display_name"`
	Capabilities ServiceTypeCapabilities `json:"capabilities"`
}

var serviceTypeDefinitions = []ServiceTypeDefinition{{
	ID: ServiceTypeVLESSReality, DisplayName: "VLESS REALITY",
	Capabilities: ServiceTypeCapabilities{
		XrayInbound: true, ServiceControl: true, TrafficCounters: true,
		RecentActivity: true, DynamicBlocking: true,
		SubscriptionFormats: []string{"base64", "mihomo", "sing-box"},
	},
}}

func SupportedServiceTypes() []ServiceTypeDefinition {
	values := make([]ServiceTypeDefinition, len(serviceTypeDefinitions))
	copy(values, serviceTypeDefinitions)
	for index := range values {
		values[index].Capabilities.SubscriptionFormats = append(
			[]string(nil), values[index].Capabilities.SubscriptionFormats...,
		)
	}
	return values
}

func ServiceTypeDefinitionByID(id string) (ServiceTypeDefinition, bool) {
	for _, definition := range serviceTypeDefinitions {
		if definition.ID == id {
			definition.Capabilities.SubscriptionFormats = append(
				[]string(nil), definition.Capabilities.SubscriptionFormats...,
			)
			return definition, true
		}
	}
	return ServiceTypeDefinition{}, false
}

func editableVLESSReality(value *VLESSReality) *EditableVLESSReality {
	if value == nil {
		return nil
	}
	serverName := ""
	if len(value.ServerNames) > 0 {
		serverName = value.ServerNames[0]
	}
	return &EditableVLESSReality{
		Target: value.Target, ServerName: serverName, Fingerprint: value.Fingerprint,
	}
}

func mergeServiceType(existing Service, preserveSecrets bool, editable EditableService) (Service, error) {
	if _, exists := ServiceTypeDefinitionByID(editable.Type); !exists {
		return Service{}, fmt.Errorf("unsupported service type %q", editable.Type)
	}
	switch editable.Type {
	case ServiceTypeVLESSReality:
		if editable.VLESSReality == nil {
			return Service{}, fmt.Errorf("vless_reality: configuration is required")
		}
		privateKey := ""
		var shortIDs []string
		if preserveSecrets && existing.VLESSReality != nil {
			privateKey = existing.VLESSReality.PrivateKey
			shortIDs = append([]string(nil), existing.VLESSReality.ShortIDs...)
		} else {
			var err error
			privateKey, shortIDs, err = newVLESSRealitySecrets()
			if err != nil {
				return Service{}, err
			}
		}
		existing.VLESSReality = &VLESSReality{
			Target: editable.VLESSReality.Target, ServerNames: []string{editable.VLESSReality.ServerName},
			PrivateKey: privateKey, ShortIDs: shortIDs, Flow: VLESSVisionFlow,
			Fingerprint: editable.VLESSReality.Fingerprint,
		}
		return existing, nil
	default:
		return Service{}, fmt.Errorf("unsupported service type %q", editable.Type)
	}
}

func validateServiceType(service Service, field string) error {
	switch service.Type {
	case ServiceTypeVLESSReality:
		if service.VLESSReality == nil {
			return fmt.Errorf("%s.vless_reality: configuration is required", field)
		}
		return validateVLESSReality(*service.VLESSReality, field+".vless_reality")
	default:
		return fmt.Errorf("%s.type: unsupported service type", field)
	}
}

func validateVLESSReality(value VLESSReality, field string) error {
	targetHost, targetPort, err := net.SplitHostPort(value.Target)
	parsedTargetPort, portErr := strconv.ParseUint(targetPort, 10, 16)
	if err != nil || portErr != nil || parsedTargetPort == 0 || !validServerName(targetHost) {
		return fmt.Errorf("%s.target: must be a domain and port", field)
	}
	if len(value.ServerNames) == 0 || len(value.ServerNames) > 16 {
		return fmt.Errorf("%s.server_names: must contain 1 to 16 values", field)
	}
	seenNames := make(map[string]struct{}, len(value.ServerNames))
	for index, name := range value.ServerNames {
		if !validServerName(name) {
			return fmt.Errorf("%s.server_names[%d]: invalid server name", field, index)
		}
		if _, exists := seenNames[name]; exists {
			return fmt.Errorf("%s.server_names[%d]: duplicate server name", field, index)
		}
		seenNames[name] = struct{}{}
	}
	if _, err := RealityPublicKey(value.PrivateKey); err != nil {
		return fmt.Errorf("%s.private_key: %w", field, err)
	}
	if len(value.ShortIDs) == 0 || len(value.ShortIDs) > 16 {
		return fmt.Errorf("%s.short_ids: must contain 1 to 16 values", field)
	}
	seenShortIDs := make(map[string]struct{}, len(value.ShortIDs))
	for index, shortID := range value.ShortIDs {
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
	if value.Flow != VLESSVisionFlow {
		return fmt.Errorf("%s.flow: must be %q", field, VLESSVisionFlow)
	}
	switch value.Fingerprint {
	case "chrome", "firefox", "safari", "ios", "android", "edge", "random", "randomized":
	default:
		return fmt.Errorf("%s.fingerprint: unsupported fingerprint", field)
	}
	return nil
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

func newVLESSRealitySecrets() (string, []string, error) {
	privateKey := make([]byte, 32)
	shortID := make([]byte, 8)
	for _, value := range [][]byte{privateKey, shortID} {
		if _, err := rand.Read(value); err != nil {
			return "", nil, fmt.Errorf("generate configuration secret: %w", err)
		}
	}
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64
	return base64.RawURLEncoding.EncodeToString(privateKey), []string{hex.EncodeToString(shortID)}, nil
}

func cloneVLESSReality(value *VLESSReality) *VLESSReality {
	if value == nil {
		return nil
	}
	clone := *value
	clone.ServerNames = append([]string(nil), value.ServerNames...)
	clone.ShortIDs = append([]string(nil), value.ShortIDs...)
	return &clone
}

func validServerName(value string) bool {
	if _, err := netip.ParseAddr(value); err == nil {
		return false
	}
	return len(value) <= 253 && domainPattern.MatchString(value)
}
