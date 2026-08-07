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
	value.DNS = config.DNSConfiguration{
		Enabled: true, QueryStrategy: config.DNSQueryStrategyUseIPv4,
		Servers: []config.DNSServer{
			{
				ServerID: "regional", DisplayName: "Regional DNS", Enabled: true,
				Transport: config.DNSTransportUDP, Address: "1.1.1.1", Port: 53,
				Domains: []string{"example.com"},
			},
			{ServerID: "system", DisplayName: "System DNS", Enabled: true, Transport: config.DNSTransportSystem},
			{
				ServerID: "disabled", DisplayName: "Disabled DNS", Enabled: false,
				Transport: config.DNSTransportDoH, Address: "https://dns.google/dns-query",
			},
		},
	}
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
			Settings struct {
				DomainStrategy string `json:"domainStrategy"`
			} `json:"settings"`
		} `json:"outbounds"`
		DNS struct {
			QueryStrategy          string `json:"queryStrategy"`
			DisableFallbackIfMatch bool   `json:"disableFallbackIfMatch"`
			Servers                []struct {
				Address       string   `json:"address"`
				Port          uint16   `json:"port"`
				Domains       []string `json:"domains"`
				QueryStrategy string   `json:"queryStrategy"`
				SkipFallback  bool     `json:"skipFallback"`
			} `json:"servers"`
		} `json:"dns"`
		Policy struct {
			Levels map[string]struct {
				StatsUserOnline bool `json:"statsUserOnline"`
			} `json:"levels"`
		} `json:"policy"`
		Routing struct {
			DomainStrategy string `json:"domainStrategy"`
			Rules          []struct {
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
		generated.Outbounds[0].Settings.DomainStrategy != "UseIPv4" ||
		!generated.Policy.Levels["0"].StatsUserOnline {
		t.Fatalf("generated Xray configuration = %+v", generated)
	}
	if len(generated.Routing.Rules) != 3 || generated.Routing.Rules[0].RuleTag != APIRuleTag ||
		generated.Routing.DomainStrategy != "IPIfNonMatch" ||
		generated.Routing.Rules[1].RuleTag != "relayward-static-block-private" ||
		generated.Routing.Rules[1].IP[0] != "192.0.2.0/24" ||
		generated.Routing.Rules[2].Domain[0] != "domain:example.com" ||
		generated.Routing.Rules[2].Protocol[0] != "tls" || len(generated.Routing.Rules[2].InboundTag) != 2 {
		t.Fatalf("generated routing configuration = %+v", generated.Routing)
	}
	if generated.DNS.QueryStrategy != "UseIPv4" || !generated.DNS.DisableFallbackIfMatch ||
		len(generated.DNS.Servers) != 2 || generated.DNS.Servers[0].Address != "1.1.1.1" ||
		generated.DNS.Servers[0].Port != 53 || generated.DNS.Servers[0].Domains[0] != "domain:example.com" ||
		!generated.DNS.Servers[0].SkipFallback || generated.DNS.Servers[1].Address != "localhost" ||
		generated.DNS.Servers[1].QueryStrategy != "UseIPv4" {
		t.Fatalf("generated DNS configuration = %+v", generated.DNS)
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
		DNS       json.RawMessage `json:"dns"`
		Outbounds []struct {
			Settings struct {
				DomainStrategy string `json:"domainStrategy"`
			} `json:"settings"`
		} `json:"outbounds"`
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
	if len(generated.DNS) != 0 || generated.Outbounds[0].Settings.DomainStrategy != "" {
		t.Fatal("disabled DNS unexpectedly changed the Xray configuration")
	}
}

func TestRenderDNSLocalTransports(t *testing.T) {
	t.Parallel()
	value := testConfiguration(t)
	value.DNS = config.DNSConfiguration{
		Enabled: true, QueryStrategy: config.DNSQueryStrategyUseIPv6,
		Servers: []config.DNSServer{
			{ServerID: "tcp", DisplayName: "TCP", Enabled: true, Transport: config.DNSTransportTCP, Address: "2001:4860:4860::8888", Port: 53},
			{ServerID: "doh", DisplayName: "DoH", Enabled: true, Transport: config.DNSTransportDoH, Address: "https://dns.google/dns-query"},
		},
	}
	raw, err := Render(value)
	if err != nil {
		t.Fatal(err)
	}
	var generated struct {
		DNS struct {
			Servers []struct {
				Address string `json:"address"`
			} `json:"servers"`
		} `json:"dns"`
	}
	if err := json.Unmarshal(raw, &generated); err != nil {
		t.Fatal(err)
	}
	if len(generated.DNS.Servers) != 2 || generated.DNS.Servers[0].Address != "tcp+local://[2001:4860:4860::8888]" ||
		generated.DNS.Servers[1].Address != "https+local://dns.google/dns-query" {
		t.Fatalf("generated DNS transports = %+v", generated.DNS.Servers)
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
			Listen: "0.0.0.0", Port: 443, PublicHost: "edge.example.com", PublicPort: 443,
			VLESSReality: &config.EditableVLESSReality{
				Target: "www.microsoft.com:443", ServerName: "www.microsoft.com", Fingerprint: "chrome",
			},
		},
		{
			Type: config.ServiceTypeVLESSReality, Enabled: true, ServiceID: "reality-backup", DisplayName: "Reality Backup",
			Listen: "0.0.0.0", Port: 8443, PublicHost: "backup.example.com", PublicPort: 8443,
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
