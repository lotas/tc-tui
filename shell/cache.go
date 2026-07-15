package shell

import (
	"time"

	"github.com/taskcluster/tc-tui/resource"
)

// cacheKey identifies one (resource, scope, facet) list result. scope is ""
// for an unscoped List(), and facet is "" for anything other than a
// ServerFaceted FacetList/FacetCounts pair.
type cacheKey struct {
	resource string
	scope    string
	facet    string
}

func cacheKeyFor(res resource.Resource, scope, facetValue string) cacheKey {
	return cacheKey{resource: res.Name(), scope: scope, facet: facetValue}
}

// cacheEntry holds one cached list fetch. counts is only populated for
// ServerFaceted entries (FacetCounts is fetched paired with FacetList in
// loadList, so it rides along in the same entry rather than getting its own
// cache); it is nil otherwise. subtitle is similarly only populated for a
// ScopeSubtitle resource; "" otherwise.
type cacheEntry struct {
	rows      []resource.Row
	counts    map[string]int
	subtitle  string
	fetchedAt time.Time
}

// listCache is a short-lived, session-lifetime cache of list/facet fetches,
// keyed by (resource, scope, facet). Entries are never evicted outright —
// staleness is checked entirely by TTL at read time in get(), and the
// working set of distinct keys in one TUI session is small enough that this
// is fine.
type listCache struct {
	entries map[cacheKey]cacheEntry
}

func newListCache() *listCache {
	return &listCache{entries: make(map[cacheKey]cacheEntry)}
}

// get returns the cached entry for key if one exists and is still within
// ttl. ttl <= 0 (a resource with auto-refresh disabled) always misses —
// mirroring RefreshInterval's existing meaning of "don't apply a freshness
// cadence to this resource" so the cache doesn't silently serve old data for
// it either.
func (c *listCache) get(key cacheKey, ttl time.Duration) (cacheEntry, bool) {
	if ttl <= 0 {
		return cacheEntry{}, false
	}

	entry, ok := c.entries[key]
	if !ok || time.Since(entry.fetchedAt) >= ttl {
		return cacheEntry{}, false
	}

	return entry, true
}

func (c *listCache) set(key cacheKey, entry cacheEntry) {
	c.entries[key] = entry
}
