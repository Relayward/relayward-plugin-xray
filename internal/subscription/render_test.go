package subscription

import (
	"bytes"
	"testing"

	centerpluginv1 "github.com/Relayward/relayward-sdk/centerplugin/v1"

	"github.com/Relayward/relayward-plugin-xray/internal/config"
)

func TestRenderMultipleServicesInAllFormatsWithStableCredentials(t *testing.T) {
	t.Parallel()
	configuration := testConfiguration(t)
	request := &centerpluginv1.RenderSubscriptionRequest{
		AuthorizationId: "10000000-0000-4000-8000-000000000001",
		NodeId:          "20000000-0000-4000-8000-000000000002", PublicAddress: "edge.example.com",
		Services: []*centerpluginv1.SubscriptionServiceBinding{
			{ServiceId: "reality-backup", DisplayName: "Edge Backup"},
			{ServiceId: "reality-main", DisplayName: "Edge Main"},
		},
	}
	first, err := Render(configuration, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Render(configuration, request)
	if err != nil {
		t.Fatal(err)
	}
	if err := centerpluginv1.ValidateRenderSubscriptionResponse(request, first); err != nil {
		t.Fatal(err)
	}
	if len(first.Services) != 2 || len(first.Services[0].Uris) != 1 || len(first.Services[1].Uris) != 1 ||
		first.Services[0].Uris[0] != second.Services[0].Uris[0] ||
		first.Services[0].Uris[0] == first.Services[1].Uris[0] {
		t.Fatalf("rendered subscription = %+v", first)
	}
	if !bytes.Contains(first.Services[0].MihomoProxiesJson[0], []byte(`"server":"edge.example.com"`)) ||
		!bytes.Contains(first.Services[0].SingBoxOutboundsJson[0], []byte(`"server_port":9443`)) ||
		!bytes.Contains(first.Services[1].SingBoxOutboundsJson[0], []byte(`"server_port":8443`)) {
		t.Fatalf("rendered fragments = %+v", first.Services)
	}
}

func TestRenderRejectsMissingPublicAddressAndUnknownService(t *testing.T) {
	t.Parallel()
	configuration := testConfiguration(t)
	request := &centerpluginv1.RenderSubscriptionRequest{
		AuthorizationId: "10000000-0000-4000-8000-000000000001",
		NodeId:          "20000000-0000-4000-8000-000000000002",
		Services: []*centerpluginv1.SubscriptionServiceBinding{{
			ServiceId: "reality-main", DisplayName: "Edge VLESS",
		}},
	}
	if _, err := Render(configuration, request); err == nil {
		t.Fatal("Render() accepted a missing public address")
	}
	request.PublicAddress = "edge.example.com"
	request.Services[0].ServiceId = "unknown"
	if _, err := Render(configuration, request); err == nil {
		t.Fatal("Render() accepted an unknown service")
	}
}

func TestSupportedFormatsMatchVLESSRealityRenderer(t *testing.T) {
	t.Parallel()
	formats := SupportedFormats(config.ServiceTypeVLESSReality)
	if !SupportsServiceType(config.ServiceTypeVLESSReality) || len(formats) != 3 ||
		formats[0] != "base64" || formats[1] != "mihomo" || formats[2] != "sing-box" ||
		SupportsServiceType("unknown") || SupportedFormats("unknown") != nil {
		t.Fatalf("supported formats = %v", formats)
	}
}

func testConfiguration(t *testing.T) config.Configuration {
	t.Helper()
	value, err := config.NewConfiguration("26.3.27", 10085, []config.EditableService{
		{
			Type: config.ServiceTypeVLESSReality, Enabled: true, ServiceID: "reality-main", DisplayName: "Reality Main",
			Listen: "0.0.0.0", Port: 443, PublicPort: 8443,
			VLESSReality: &config.EditableVLESSReality{
				Target: "www.microsoft.com:443", ServerName: "www.microsoft.com", Fingerprint: "chrome",
			},
		},
		{
			Type: config.ServiceTypeVLESSReality, Enabled: true, ServiceID: "reality-backup", DisplayName: "Reality Backup",
			Listen: "0.0.0.0", Port: 444, PublicPort: 9443,
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
