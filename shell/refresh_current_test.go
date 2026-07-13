package shell

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/taskcluster/tc-tui/resource"
)

func TestRefreshCurrentNoOpOnEmptyStack(t *testing.T) {
	s := New(resource.NewRegistry())

	s.refreshCurrent() // must not panic
}

func TestRefreshCurrentBypassesListCache(t *testing.T) {
	s := New(resource.NewRegistry())
	res := &fakeServerFacetedListResource{
		fakeResource: fakeResource{name: "workers"},
		ttl:          time.Minute,
		options:      []string{"running"},
		rows: map[string][]resource.Row{
			"running": {{ID: "fresh", Cells: []string{"running"}}},
		},
	}
	s.registry.Register(res)
	s.currentListResource = "workers"
	s.currentColumns = []resource.Column{{Title: "STATE"}}
	s.currentServerFaceted = res
	s.currentFacetValue = "running"
	s.stack.Push(View{ResourceName: "workers", Kind: ListKind, Scope: "pool-a"})
	s.cache.set(cacheKeyFor(res, "pool-a", "running"), cacheEntry{
		rows:      []resource.Row{{ID: "stale", Cells: []string{"running"}}},
		fetchedAt: time.Now(),
	})

	s.refreshCurrent()

	waitFor(t, func() bool {
		_, called := res.lastCall()
		return called // a cache hit would never call FacetList at all
	})
}

// fakeCountingDetailResource counts Describe calls — used to confirm
// refreshCurrent triggers a genuine re-fetch for a Detail view rather than
// no-op'ing.
type fakeCountingDetailResource struct {
	fakeResource
	calls atomic.Int32
}

func (f *fakeCountingDetailResource) Describe(id string) (resource.Detail, error) {
	f.calls.Add(1)
	return resource.Detail{}, nil
}

func TestRefreshCurrentTriggersDetailRefetch(t *testing.T) {
	s := New(resource.NewRegistry())
	res := &fakeCountingDetailResource{fakeResource: fakeResource{name: "task"}}
	s.registry.Register(res)
	s.stack.Push(View{ResourceName: "task", Kind: DetailKind, SelectedID: "task-1"})

	s.refreshCurrent()

	waitFor(t, func() bool {
		return res.calls.Load() > 0
	})
}

