package config

import "testing"

func TestDecode(t *testing.T) {
	t.Parallel()
	value, err := Decode([]byte(`{"xray_version":"26.3.27","xray_config":{"log":{"loglevel":"warning"}}}`))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if value.XrayVersion != "26.3.27" {
		t.Fatalf("XrayVersion = %q", value.XrayVersion)
	}
	if string(value.XrayConfig) != `{"log":{"loglevel":"warning"}}` {
		t.Fatalf("XrayConfig = %s", value.XrayConfig)
	}
}

func TestDecodeRejectsInvalidConfigurations(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"missing version":     `{"xray_config":{}}`,
		"leading v":           `{"xray_version":"v26.3.27","xray_config":{}}`,
		"pre-release":         `{"xray_version":"26.3.27-rc.1","xray_config":{}}`,
		"missing xray config": `{"xray_version":"26.3.27"}`,
		"null xray config":    `{"xray_version":"26.3.27","xray_config":null}`,
		"array xray config":   `{"xray_version":"26.3.27","xray_config":[]}`,
		"unknown field":       `{"xray_version":"26.3.27","xray_config":{},"token":"secret"}`,
		"trailing JSON":       `{"xray_version":"26.3.27","xray_config":{}} {}`,
	}
	for name, raw := range tests {
		name, raw := name, raw
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Decode([]byte(raw)); err == nil {
				t.Fatal("Decode() unexpectedly succeeded")
			}
		})
	}
}
