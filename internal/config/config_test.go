package config

import (
	"encoding/json"
	"testing"
)

func TestConfigurationRoundTripsTypedServicesAndStableCredentials(t *testing.T) {
	t.Parallel()
	value, err := NewConfiguration("26.3.27", 10085, testEditableServices())
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
	if decoded.XrayVersion != "26.3.27" || len(decoded.Services) != 2 || decoded.Services[0].ServiceID != "reality-backup" ||
		decoded.Services[1].ServiceID != "reality-main" {
		t.Fatalf("configuration = %+v", decoded)
	}
	first, err := DeriveCredential(decoded.CredentialSeed, "10000000-0000-4000-8000-000000000001", "reality-main")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := DeriveCredential(decoded.CredentialSeed, "10000000-0000-4000-8000-000000000001", "reality-main")
	otherService, _ := DeriveCredential(decoded.CredentialSeed, "10000000-0000-4000-8000-000000000001", "reality-backup")
	otherAuthorization, _ := DeriveCredential(decoded.CredentialSeed, "20000000-0000-4000-8000-000000000002", "reality-main")
	if first != second || first == otherService || first == otherAuthorization {
		t.Fatalf("derived credentials = %q, %q, %q, %q", first, second, otherService, otherAuthorization)
	}
	for _, service := range decoded.Services {
		publicKey, err := RealityPublicKey(service.VLESSReality.PrivateKey)
		if err != nil || len(publicKey) != 43 {
			t.Fatalf("RealityPublicKey(%q) = %q, %v", service.ServiceID, publicKey, err)
		}
	}
}

func TestSupportedServiceTypeCatalogIsDefensive(t *testing.T) {
	t.Parallel()
	definitions := SupportedServiceTypes()
	if len(definitions) != 1 || definitions[0].ID != ServiceTypeVLESSReality ||
		len(definitions[0].Capabilities.SubscriptionFormats) != 3 {
		t.Fatalf("SupportedServiceTypes() = %+v", definitions)
	}
	definitions[0].Capabilities.SubscriptionFormats[0] = "changed"
	definition, exists := ServiceTypeDefinitionByID(ServiceTypeVLESSReality)
	if !exists || definition.Capabilities.SubscriptionFormats[0] != "base64" {
		t.Fatalf("ServiceTypeDefinitionByID() = %+v, %t", definition, exists)
	}
}

func TestDecodeRejectsInvalidConfigurations(t *testing.T) {
	t.Parallel()
	value, err := NewConfiguration("26.3.27", 10085, testEditableServices())
	if err != nil {
		t.Fatal(err)
	}
	valid, _ := Encode(value)
	tests := map[string]func(*Configuration){
		"missing version":      func(value *Configuration) { value.XrayVersion = "" },
		"leading v":            func(value *Configuration) { value.XrayVersion = "v26.3.27" },
		"pre-release":          func(value *Configuration) { value.XrayVersion = "26.3.27-rc.1" },
		"privileged API":       func(value *Configuration) { value.APIPort = 80 },
		"invalid seed":         func(value *Configuration) { value.CredentialSeed = "secret" },
		"unsupported type":     func(value *Configuration) { value.Services[0].Type = "trojan" },
		"invalid service ID":   func(value *Configuration) { value.Services[0].ServiceID = "Invalid ID" },
		"duplicate service ID": func(value *Configuration) { value.Services[1].ServiceID = value.Services[0].ServiceID },
		"unsorted services": func(value *Configuration) {
			value.Services[0], value.Services[1] = value.Services[1], value.Services[0]
		},
		"missing typed config": func(value *Configuration) { value.Services[0].VLESSReality = nil },
		"invalid target": func(value *Configuration) {
			value.Services[0].VLESSReality.Target = "127.0.0.1:443"
		},
		"invalid key": func(value *Configuration) {
			value.Services[0].VLESSReality.PrivateKey = "secret"
		},
		"invalid short ID": func(value *Configuration) {
			value.Services[0].VLESSReality.ShortIDs = []string{"xyz"}
		},
		"conflicting listener": func(value *Configuration) { value.Services[1].Port = value.Services[0].Port },
		"conflicting API port": func(value *Configuration) { value.Services[0].Port = value.APIPort },
		"too many services": func(value *Configuration) {
			service := value.Services[0]
			for len(value.Services) <= MaximumServices {
				service.ServiceID += "x"
				value.Services = append(value.Services, service)
			}
		},
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
	legacy := []byte(`{"xray_version":"26.3.27","api_port":10085,"credential_seed":"secret","vless_reality":{}}`)
	if _, err := Decode(legacy); err == nil {
		t.Fatal("Decode() accepted the retired single-service configuration")
	}
}

func TestEditableConfigurationPreservesExistingSecretsAndGeneratesNewServiceSecrets(t *testing.T) {
	t.Parallel()
	value, err := NewConfiguration("26.3.27", 10085, testEditableServices()[:1])
	if err != nil {
		t.Fatal(err)
	}
	editable := Editable(value)
	editable.Services[0].DisplayName = "Updated Main"
	editable.Services = append(editable.Services, testEditableServices()[1])
	merged, err := MergeEditable(value, editable)
	if err != nil {
		t.Fatal(err)
	}
	main, mainExists := merged.FindService("reality-main")
	backup, backupExists := merged.FindService("reality-backup")
	if merged.CredentialSeed != value.CredentialSeed || !mainExists || !backupExists ||
		main.VLESSReality.PrivateKey != value.Services[0].VLESSReality.PrivateKey ||
		main.VLESSReality.ShortIDs[0] != value.Services[0].VLESSReality.ShortIDs[0] ||
		main.DisplayName != "Updated Main" {
		t.Fatalf("MergeEditable() changed protected existing configuration: %+v", merged)
	}
	if backup.VLESSReality.PrivateKey == "" || backup.VLESSReality.PrivateKey == value.Services[0].VLESSReality.PrivateKey ||
		backup.VLESSReality.ShortIDs[0] == value.Services[0].VLESSReality.ShortIDs[0] {
		t.Fatalf("MergeEditable() did not generate independent service secrets: %+v", backup)
	}
	created, err := NewFromEditable(editable)
	createdMain, createdMainExists := created.FindService("reality-main")
	if err != nil || created.CredentialSeed == "" || created.CredentialSeed == value.CredentialSeed ||
		len(created.Services) != 2 || !createdMainExists ||
		createdMain.VLESSReality.PrivateKey == value.Services[0].VLESSReality.PrivateKey {
		t.Fatalf("NewFromEditable() = %+v, %v", created, err)
	}
	deleted, err := MergeEditable(merged, EditableConfiguration{
		XrayVersion: editable.XrayVersion, APIPort: editable.APIPort, Services: editable.Services[1:],
	})
	if err != nil || len(deleted.Services) != 1 || deleted.Services[0].ServiceID != "reality-backup" ||
		deleted.Services[0].VLESSReality.PrivateKey != backup.VLESSReality.PrivateKey {
		t.Fatalf("MergeEditable(delete) = %+v, %v", deleted, err)
	}
}

func testEditableServices() []EditableService {
	return []EditableService{
		{
			Type: ServiceTypeVLESSReality, Enabled: true, ServiceID: "reality-main", DisplayName: "Reality Main",
			Listen: "0.0.0.0", Port: 443, PublicPort: 443,
			VLESSReality: &EditableVLESSReality{
				Target: "www.microsoft.com:443", ServerName: "www.microsoft.com", Fingerprint: "chrome",
			},
		},
		{
			Type: ServiceTypeVLESSReality, Enabled: true, ServiceID: "reality-backup", DisplayName: "Reality Backup",
			Listen: "0.0.0.0", Port: 8443, PublicPort: 8443,
			VLESSReality: &EditableVLESSReality{
				Target: "www.cloudflare.com:443", ServerName: "www.cloudflare.com", Fingerprint: "chrome",
			},
		},
	}
}
