package subscription

import (
	"bytes"
	"testing"

	centerpluginv1 "github.com/Relayward/relayward-sdk/centerplugin/v1"

	"github.com/Relayward/relayward-plugin-xray/internal/config"
)

func TestRenderAllFormatsWithStableCredential(t *testing.T) {
	t.Parallel()
	configuration, err := config.NewConfiguration("26.3.27", 10085, 443, 8443, "www.microsoft.com:443", "www.microsoft.com")
	if err != nil {
		t.Fatal(err)
	}
	request := &centerpluginv1.RenderSubscriptionRequest{
		AuthorizationId: "10000000-0000-4000-8000-000000000001",
		NodeId:          "20000000-0000-4000-8000-000000000002", PublicAddress: "edge.example.com",
		Services: []*centerpluginv1.SubscriptionServiceBinding{{
			ServiceId: config.VLESSRealityServiceID, DisplayName: "Edge VLESS",
		}},
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
	if len(first.Services) != 1 || len(first.Services[0].Uris) != 1 || first.Services[0].Uris[0] != second.Services[0].Uris[0] {
		t.Fatalf("rendered subscription = %+v", first)
	}
	if !bytes.Contains(first.Services[0].MihomoProxiesJson[0], []byte(`"server":"edge.example.com"`)) ||
		!bytes.Contains(first.Services[0].SingBoxOutboundsJson[0], []byte(`"server_port":8443`)) {
		t.Fatalf("rendered fragments = %+v", first.Services[0])
	}
}

func TestRenderRejectsMissingPublicAddress(t *testing.T) {
	t.Parallel()
	configuration, err := config.NewConfiguration("26.3.27", 10085, 443, 443, "www.microsoft.com:443", "www.microsoft.com")
	if err != nil {
		t.Fatal(err)
	}
	request := &centerpluginv1.RenderSubscriptionRequest{
		AuthorizationId: "10000000-0000-4000-8000-000000000001",
		NodeId:          "20000000-0000-4000-8000-000000000002",
		Services: []*centerpluginv1.SubscriptionServiceBinding{{
			ServiceId: config.VLESSRealityServiceID, DisplayName: "Edge VLESS",
		}},
	}
	if _, err := Render(configuration, request); err == nil {
		t.Fatal("Render() accepted a missing public address")
	}
}
