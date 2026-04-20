package bridge

import (
	"sync"
	"testing"
	"time"

	"arkkb/src/core/config"
)

func TestSyncQueueCoalescesPendingRequests(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})

	var seenMu sync.Mutex
	seen := []string{}

	bridge := &Bridge{}
	bridge.syncWorkspace = func(cfg *config.AppConfig) error {
		seenMu.Lock()
		seen = append(seen, cfg.LastWorkspace)
		callCount := len(seen)
		seenMu.Unlock()

		if callCount == 1 {
			close(started)
			<-release
		}
		return nil
	}
	bridge.afterWorkspaceSync = func() {}

	first := config.DefaultConfig()
	first.LastWorkspace = "one"
	bridge.enqueueSyncRequest(&SyncRequest{Snapshot: first, Reason: "first"})
	<-started

	second := config.DefaultConfig()
	second.LastWorkspace = "two"
	third := config.DefaultConfig()
	third.LastWorkspace = "three"
	bridge.enqueueSyncRequest(&SyncRequest{Snapshot: second, Reason: "second"})
	bridge.enqueueSyncRequest(&SyncRequest{Snapshot: third, Reason: "third"})
	close(release)

	deadline := time.Now().Add(2 * time.Second)
	for {
		bridge.syncMu.Lock()
		running := bridge.syncRunning
		pending := bridge.syncPending
		bridge.syncMu.Unlock()
		if !running && pending == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for sync queue to drain")
		}
		time.Sleep(10 * time.Millisecond)
	}

	seenMu.Lock()
	defer seenMu.Unlock()
	if len(seen) != 2 {
		t.Fatalf("expected 2 sync executions, got %d (%v)", len(seen), seen)
	}
	if seen[0] != "one" {
		t.Fatalf("expected first sync to use initial snapshot, got %s", seen[0])
	}
	if seen[1] != "three" {
		t.Fatalf("expected queued sync to use latest snapshot, got %s", seen[1])
	}
}
