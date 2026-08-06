package xrayruntime

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
)

func TestOnlineIPResponseDecodesOfficialMapWireFormat(t *testing.T) {
	t.Parallel()
	entry := protowire.AppendTag(nil, 1, protowire.BytesType)
	entry = protowire.AppendString(entry, "2001:db8::1")
	entry = protowire.AppendTag(entry, 2, protowire.VarintType)
	entry = protowire.AppendVarint(entry, 1786017600)
	raw := protowire.AppendTag(nil, 1, protowire.BytesType)
	raw = protowire.AppendString(raw, "user>>>relayward:test:vless-reality>>>online")
	raw = protowire.AppendTag(raw, 2, protowire.BytesType)
	raw = protowire.AppendBytes(raw, entry)
	response := &getStatsOnlineIPListResponse{}
	if err := proto.Unmarshal(raw, protoadapt.MessageV2Of(response)); err != nil {
		t.Fatal(err)
	}
	if response.IPs["2001:db8::1"] != 1786017600 {
		t.Fatalf("decoded response = %+v", response)
	}
}
