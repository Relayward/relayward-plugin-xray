package xrayruntime

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestTelemetryStorePersistsCursorAndRefreshesOnlineActivity(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	store, err := openTelemetryStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	streamID := store.streamID()
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	active := testActivitySources("192.0.2.1")
	first, err := store.appendSnapshot(0, 10, active, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Events) != 1 || first.Events[0].Sequence != 1 || first.NextSequence != 1 || first.HasMore {
		t.Fatalf("first page = %+v", first)
	}
	replayed, err := store.appendSnapshot(0, 10, active, now.Add(time.Second))
	if err != nil || len(replayed.Events) != 1 || replayed.Events[0] != first.Events[0] {
		t.Fatalf("replayed page = %+v, %v", replayed, err)
	}
	acknowledged, err := store.appendSnapshot(1, 10, active, now.Add(time.Second))
	if err != nil || len(acknowledged.Events) != 0 || acknowledged.NextSequence != 1 {
		t.Fatalf("acknowledged page = %+v, %v", acknowledged, err)
	}
	refreshed, err := store.appendSnapshot(1, 10, active, now.Add(activityRefreshInterval+time.Second))
	if err != nil || len(refreshed.Events) != 1 || refreshed.Events[0].Sequence != 2 {
		t.Fatalf("refreshed page = %+v, %v", refreshed, err)
	}
	reopened, err := openTelemetryStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.streamID() != streamID {
		t.Fatalf("stream ID changed from %q to %q", streamID, reopened.streamID())
	}
	persisted, err := reopened.appendSnapshot(1, 10, active, now.Add(activityRefreshInterval+2*time.Second))
	if err != nil || len(persisted.Events) != 1 || persisted.Events[0].Sequence != 2 {
		t.Fatalf("persisted page = %+v, %v", persisted, err)
	}
	if _, err := reopened.appendSnapshot(0, 10, active, now.Add(time.Minute)); !errors.Is(err, ErrTelemetryDataLoss) {
		t.Fatalf("old cursor error = %v", err)
	}
}

func TestTelemetryStoreRejectsOverflowWithoutPartialState(t *testing.T) {
	t.Parallel()
	store, err := openTelemetryStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	active := make(map[string]ActivitySource, maximumQueuedActivity+1)
	for index := 1; index <= maximumQueuedActivity+1; index++ {
		ip := fmt.Sprintf("2001:db8::%x", index)
		for key, value := range testActivitySources(ip) {
			active[key] = value
		}
	}
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	if _, err := store.appendSnapshot(0, 10, active, now); !errors.Is(err, ErrTelemetryFull) {
		t.Fatalf("overflow error = %v", err)
	}
	page, err := store.appendSnapshot(0, 10, testActivitySources("192.0.2.2"), now)
	if err != nil || len(page.Events) != 1 || page.Events[0].Sequence != 1 {
		t.Fatalf("page after overflow = %+v, %v", page, err)
	}
}

func testActivitySources(sourceIP string) map[string]ActivitySource {
	value := ActivitySource{
		AuthorizationID: "10000000-0000-4000-8000-000000000001",
		ServiceID:       "vless-reality",
		SourceIP:        sourceIP,
	}
	return map[string]ActivitySource{serviceKey(value.AuthorizationID, value.ServiceID) + "\x00" + sourceIP: value}
}
