package xrayconfig

import (
	"encoding/json"
	"testing"

	"github.com/Relayward/relayward-plugin-xray/internal/config"
)

func TestRenderBuildsTypedServiceInbounds(t *testing.T) {
	t.Parallel()
	value := testConfiguration(t)
	value.Routing = config.RoutingConfiguration{Rules: []config.RoutingRule{
		{
			RuleID: "block-private", DisplayName: "Block private", Enabled: true,
			IPCIDRs: []string{"192.0.2.0/24"}, Action: config.RoutingActionBlocked,
		},
		{
			RuleID: "allow-example", DisplayName: "Allow example", Enabled: true,
			Domains: []string{"example.com"}, Protocols: []string{"tls"}, Action: config.RoutingActionDirect,
		},
	}}
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
			Sniffing struct {
				Enabled      bool     `json:"enabled"`
				DestOverride []string `json:"destOverride"`
				RouteOnly    bool     `json:"routeOnly"`
			} `json:"sniffing"`
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
		Routing struct {
			Rules []struct {
				RuleTag     string   `json:"ruleTag"`
				OutboundTag string   `json:"outboundTag"`
				Domain      []string `json:"domain"`
				IP          []string `json:"ip"`
				Protocol    []string `json:"protocol"`
				InboundTag  []string `json:"inboundTag"`
			} `json:"rules"`
		} `json:"routing"`
	}
	if err := json.Unmarshal(raw, &generated); err != nil {
		t.Fatal(err)
	}
	if len(generated.API.Services) != 3 || generated.API.Services[1] != "RoutingService" ||
		len(generated.Inbounds) != 3 || generated.Inbounds[1].Tag != "reality-backup" ||
		generated.Inbounds[1].Protocol != "vless" ||
		generated.Inbounds[1].StreamSettings.RealitySettings.Target != "www.cloudflare.com:443" ||
		!generated.Inbounds[1].Sniffing.Enabled || !generated.Inbounds[1].Sniffing.RouteOnly ||
		len(generated.Inbounds[1].Sniffing.DestOverride) != 3 ||
		generated.Inbounds[2].Tag != "reality-main" || len(generated.Outbounds) != 2 ||
		generated.Outbounds[1].Tag != "blocked" || generated.Outbounds[1].Protocol != "blackhole" ||
		!generated.Policy.Levels["0"].StatsUserOnline {
		t.Fatalf("generated Xray configuration = %+v", generated)
	}
	if len(generated.Routing.Rules) != 3 || generated.Routing.Rules[0].RuleTag != APIRuleTag ||
		generated.Routing.Rules[1].RuleTag != "relayward-static-block-private" ||
		generated.Routing.Rules[1].IP[0] != "192.0.2.0/24" ||
		generated.Routing.Rules[2].Domain[0] != "domain:example.com" ||
		generated.Routing.Rules[2].Protocol[0] != "tls" || len(generated.Routing.Rules[2].InboundTag) != 2 {
		t.Fatalf("generated routing configuration = %+v", generated.Routing)
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
		Inbounds []struct {
			Sniffing json.RawMessage `json:"sniffing"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(raw, &generated); err != nil {
		t.Fatal(err)
	}
	if len(generated.Inbounds) != 2 {
		t.Fatalf("inbounds = %d, want API plus one enabled service", len(generated.Inbounds))
	}
	if len(generated.Inbounds[1].Sniffing) != 0 {
		t.Fatal("disabled or absent domain routing unexpectedly enabled sniffing")
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
