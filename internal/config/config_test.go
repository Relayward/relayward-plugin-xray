package config

import (
	"encoding/json"
	"testing"
)

func TestConfigurationBuildsXrayAndStableCredentials(t *testing.T) {
	t.Parallel()
	value, err := NewConfiguration("26.3.27", 10085, 443, 443, "www.microsoft.com:443", "www.microsoft.com")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(raw)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if decoded.XrayVersion != "26.3.27" || decoded.VLESSReality.ServiceID != VLESSRealityServiceID {
		t.Fatalf("configuration = %+v", decoded)
	}
	xrayJSON, err := decoded.XrayJSON()
	if err != nil || !json.Valid(xrayJSON) {
		t.Fatalf("XrayJSON() = %s, %v", xrayJSON, err)
	}
	var generated struct {
		API struct {
			Services []string `json:"services"`
		} `json:"api"`
		Outbounds []struct {
			Tag      string `json:"tag"`
			Protocol string `json:"protocol"`
		} `json:"outbounds"`
		Policy struct {
			Levels map[string]struct {
				StatsUserOnline bool `json:"statsUserOnline"`
			} `json:"levels"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(xrayJSON, &generated); err != nil {
		t.Fatal(err)
	}
	if len(generated.API.Services) != 3 || generated.API.Services[1] != "RoutingService" ||
		len(generated.Outbounds) != 2 || generated.Outbounds[1].Tag != "blocked" ||
		generated.Outbounds[1].Protocol != "blackhole" || !generated.Policy.Levels["0"].StatsUserOnline {
		t.Fatalf("generated Xray services = %+v", generated)
	}
	first, err := DeriveCredential(decoded.CredentialSeed, "10000000-0000-4000-8000-000000000001", VLESSRealityServiceID)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := DeriveCredential(decoded.CredentialSeed, "10000000-0000-4000-8000-000000000001", VLESSRealityServiceID)
	other, _ := DeriveCredential(decoded.CredentialSeed, "20000000-0000-4000-8000-000000000002", VLESSRealityServiceID)
	if first != second || first == other {
		t.Fatalf("derived credentials = %q, %q, %q", first, second, other)
	}
	publicKey, err := RealityPublicKey(decoded.VLESSReality.PrivateKey)
	if err != nil || len(publicKey) != 43 {
		t.Fatalf("RealityPublicKey() = %q, %v", publicKey, err)
	}
}

func TestDecodeRejectsInvalidConfigurations(t *testing.T) {
	t.Parallel()
	value, err := NewConfiguration("26.3.27", 10085, 443, 443, "www.microsoft.com:443", "www.microsoft.com")
	if err != nil {
		t.Fatal(err)
	}
	valid, _ := Encode(value)
	tests := map[string]func(*Configuration){
		"missing version":  func(value *Configuration) { value.XrayVersion = "" },
		"leading v":        func(value *Configuration) { value.XrayVersion = "v26.3.27" },
		"pre-release":      func(value *Configuration) { value.XrayVersion = "26.3.27-rc.1" },
		"privileged API":   func(value *Configuration) { value.APIPort = 80 },
		"invalid seed":     func(value *Configuration) { value.CredentialSeed = "secret" },
		"invalid service":  func(value *Configuration) { value.VLESSReality.ServiceID = "other" },
		"invalid target":   func(value *Configuration) { value.VLESSReality.Target = "127.0.0.1:443" },
		"invalid key":      func(value *Configuration) { value.VLESSReality.PrivateKey = "secret" },
		"invalid short ID": func(value *Configuration) { value.VLESSReality.ShortIDs = []string{"xyz"} },
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var candidate Configuration
			if err := json.Unmarshal(valid, &candidate); err != nil {
				t.Fatal(err)
			}
			mutate(&candidate)
			raw, _ := json.Marshal(candidate)
			if _, err := Decode(raw); err == nil {
				t.Fatal("Decode() unexpectedly succeeded")
			}
		})
	}
	if _, err := Decode(append(valid, []byte(` {}`)...)); err == nil {
		t.Fatal("Decode() accepted trailing JSON")
	}
	var object map[string]any
	_ = json.Unmarshal(valid, &object)
	object["unknown"] = true
	raw, _ := json.Marshal(object)
	if _, err := Decode(raw); err == nil {
		t.Fatal("Decode() accepted an unknown field")
	}
}

func TestEditableConfigurationPreservesSecrets(t *testing.T) {
	t.Parallel()
	value, err := NewConfiguration("26.3.27", 10085, 443, 443, "www.microsoft.com:443", "www.microsoft.com")
	if err != nil {
		t.Fatal(err)
	}
	editable := Editable(value)
	editable.VLESSReality.DisplayName = "Edge VLESS"
	merged, err := MergeEditable(value, editable)
	if err != nil {
		t.Fatal(err)
	}
	if merged.CredentialSeed != value.CredentialSeed || merged.VLESSReality.PrivateKey != value.VLESSReality.PrivateKey ||
		merged.VLESSReality.ShortIDs[0] != value.VLESSReality.ShortIDs[0] || merged.VLESSReality.DisplayName != "Edge VLESS" {
		t.Fatalf("MergeEditable() changed protected configuration: %+v", merged)
	}
	created, err := NewFromEditable(editable)
	if err != nil || created.CredentialSeed == "" || created.VLESSReality.PrivateKey == "" {
		t.Fatalf("NewFromEditable() = %+v, %v", created, err)
	}
}
