package xrayconfig

import (
	"encoding/json"
	"testing"

	"github.com/Relayward/relayward-plugin-xray/internal/config"
)

func TestRenderBuildsTypedServiceInbounds(t *testing.T) {
	t.Parallel()
	value := testConfiguration(t)
	raw, err := Render(value)
	if err != nil || !json.Valid(raw) {
		t.Fatalf("Render() = %s, %v", raw, err)
	}
	var generated struct {
		API struct {
			Services []string `json:"services"`
		} `json:"api"`
		Inbounds []struct {
			Tag            string `json:"tag"`
			Protocol       string `json:"protocol"`
			StreamSettings struct {
				RealitySettings struct {
					Target string `json:"target"`
				} `json:"realitySettings"`
			} `json:"streamSettings"`
		} `json:"inbounds"`
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
	if err := json.Unmarshal(raw, &generated); err != nil {
		t.Fatal(err)
	}
	if len(generated.API.Services) != 3 || generated.API.Services[1] != "RoutingService" ||
		len(generated.Inbounds) != 3 || generated.Inbounds[1].Tag != "reality-backup" ||
		generated.Inbounds[1].Protocol != "vless" ||
		generated.Inbounds[1].StreamSettings.RealitySettings.Target != "www.cloudflare.com:443" ||
		generated.Inbounds[2].Tag != "reality-main" || len(generated.Outbounds) != 2 ||
		generated.Outbounds[1].Tag != "blocked" || generated.Outbounds[1].Protocol != "blackhole" ||
		!generated.Policy.Levels["0"].StatsUserOnline {
		t.Fatalf("generated Xray configuration = %+v", generated)
	}
}

func TestRenderOmitsDisabledServices(t *testing.T) {
	t.Parallel()
	value := testConfiguration(t)
	value.Services[1].Enabled = false
	raw, err := Render(value)
	if err != nil {
		t.Fatal(err)
	}
	var generated struct {
		Inbounds []json.RawMessage `json:"inbounds"`
	}
	if err := json.Unmarshal(raw, &generated); err != nil {
		t.Fatal(err)
	}
	if len(generated.Inbounds) != 2 {
		t.Fatalf("inbounds = %d, want API plus one enabled service", len(generated.Inbounds))
	}
}

func TestSupportsOnlyRegisteredServiceTypes(t *testing.T) {
	t.Parallel()
	if !SupportsServiceType(config.ServiceTypeVLESSReality) || SupportsServiceType("unknown") {
		t.Fatal("Xray renderer service type support is inconsistent")
	}
}

func testConfiguration(t *testing.T) config.Configuration {
	t.Helper()
	value, err := config.NewConfiguration("26.3.27", 10085, []config.EditableService{
		{
			Type: config.ServiceTypeVLESSReality, Enabled: true, ServiceID: "reality-main", DisplayName: "Reality Main",
			Listen: "0.0.0.0", Port: 443, PublicPort: 443,
			VLESSReality: &config.EditableVLESSReality{
				Target: "www.microsoft.com:443", ServerName: "www.microsoft.com", Fingerprint: "chrome",
			},
		},
		{
			Type: config.ServiceTypeVLESSReality, Enabled: true, ServiceID: "reality-backup", DisplayName: "Reality Backup",
			Listen: "0.0.0.0", Port: 8443, PublicPort: 8443,
			VLESSReality: &config.EditableVLESSReality{
				Target: "www.cloudflare.com:443", ServerName: "www.cloudflare.com", Fingerprint: "chrome",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return value
}
