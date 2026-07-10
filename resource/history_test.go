package resource

import (
	"fmt"
	"testing"
	"time"
)

func TestHistoryResourceRecordDedupesAndMovesToTop(t *testing.T) {
	h := NewHistoryResource()

	first := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	second := time.Date(2026, 7, 10, 10, 5, 0, 0, time.UTC)

	h.Record(HistoryEntry{ResourceName: "workerpools", Kind: 1, SelectedID: "pool-a", Title: "Worker Pool :: pool-a", VisitedAt: first})
	h.Record(HistoryEntry{ResourceName: "roles", Kind: 1, SelectedID: "role-a", Title: "Role :: role-a", VisitedAt: first})
	h.Record(HistoryEntry{ResourceName: "workerpools", Kind: 1, SelectedID: "pool-a", Title: "Worker Pool :: pool-a", VisitedAt: second})

	entries := h.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected dedup to leave 2 entries, got %d: %+v", len(entries), entries)
	}
	if entries[0].ResourceName != "workerpools" || !entries[0].VisitedAt.Equal(second) {
		t.Fatalf("expected the re-visited pool-a entry at the top with the newer timestamp, got %+v", entries[0])
	}
	if entries[1].ResourceName != "roles" {
		t.Fatalf("expected roles entry second, got %+v", entries[1])
	}
}

func TestHistoryResourceRecordEvictsOldestPastCap(t *testing.T) {
	h := NewHistoryResource()

	for i := 0; i < historyMaxEntries+5; i++ {
		h.Record(HistoryEntry{
			ResourceName: "workers",
			Kind:         1,
			SelectedID:   fmt.Sprintf("worker-%d", i),
			VisitedAt:    time.Now(),
		})
	}

	entries := h.Entries()
	if len(entries) != historyMaxEntries {
		t.Fatalf("expected cap of %d entries, got %d", historyMaxEntries, len(entries))
	}
	if entries[0].SelectedID != fmt.Sprintf("worker-%d", historyMaxEntries+4) {
		t.Fatalf("expected the most recently added entry at the top, got %+v", entries[0])
	}
}

func TestHistoryResourceListBuildsNavTargetForDetailEntry(t *testing.T) {
	h := NewHistoryResource()
	h.Record(HistoryEntry{ResourceName: "workerpools", Kind: 1, SelectedID: "pool-a", Title: "Worker Pool :: pool-a", VisitedAt: time.Now()})

	rows, err := h.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if row.NavTarget == nil || row.NavTarget.Kind != NavDetail || row.NavTarget.ResourceName != "workerpools" || row.NavTarget.ID != "pool-a" {
		t.Fatalf("expected a NavDetail target at (workerpools, pool-a), got %+v", row.NavTarget)
	}
	if row.Cells[0] != "workerpools" || row.Cells[1] != "pool-a" || row.Cells[3] != "Worker Pool :: pool-a" {
		t.Fatalf("unexpected cells: %+v", row.Cells)
	}
}

func TestHistoryResourceListBuildsNavTargetForScopedListEntry(t *testing.T) {
	h := NewHistoryResource()
	h.Record(HistoryEntry{ResourceName: "workers", Kind: 0, Scope: "pool-a", VisitedAt: time.Now()})

	rows, err := h.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if row.NavTarget == nil || row.NavTarget.Kind != NavScopedList || row.NavTarget.ResourceName != "workers" || row.NavTarget.ID != "pool-a" {
		t.Fatalf("expected a NavScopedList target at (workers, pool-a), got %+v", row.NavTarget)
	}
	if row.Cells[1] != "pool-a" || row.Cells[3] != "" {
		t.Fatalf("expected scope as the ID cell and a blank title cell, got %+v", row.Cells)
	}
}

func TestHistoryResourceRestoreReplacesEntriesAndTruncatesToCap(t *testing.T) {
	h := NewHistoryResource()
	h.Record(HistoryEntry{ResourceName: "roles", Kind: 1, SelectedID: "role-a", VisitedAt: time.Now()})

	oversized := make([]HistoryEntry, historyMaxEntries+3)
	for i := range oversized {
		oversized[i] = HistoryEntry{ResourceName: "workers", Kind: 1, SelectedID: fmt.Sprintf("worker-%d", i), VisitedAt: time.Now()}
	}

	h.Restore(oversized)

	entries := h.Entries()
	if len(entries) != historyMaxEntries {
		t.Fatalf("expected Restore to truncate to %d entries, got %d", historyMaxEntries, len(entries))
	}
	if entries[0].SelectedID != "worker-0" {
		t.Fatalf("expected Restore to trust the given order, got %+v", entries[0])
	}
}

func TestHistoryResourceDescribeReturnsError(t *testing.T) {
	h := NewHistoryResource()
	if _, err := h.Describe("anything"); err == nil {
		t.Fatalf("expected Describe to return an error")
	}
}
