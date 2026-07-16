package taskcluster

// paginate repeatedly calls fetch with the continuation token from the
// previous call, collecting all pages, until fetch returns an empty token.
func paginate[T any](fetch func(continuationToken string) ([]T, string, error)) ([]T, error) {
	items, _, err := paginateUpTo(0, fetch)
	return items, err
}

// paginateUpTo is paginate with a safety cap: it stops requesting further
// pages once at least limit items have been collected, reporting via
// truncated whether pages were left unfetched (i.e. a continuation token
// still remained when it stopped). Items already fetched past the limit are
// kept, not trimmed — the caller paid for them, and an honest "N+" display
// wants the real N. limit <= 0 means no cap (never truncated).
func paginateUpTo[T any](limit int, fetch func(continuationToken string) ([]T, string, error)) ([]T, bool, error) {
	items := make([]T, 0)
	cont := ""

	for {
		page, next, err := fetch(cont)
		if err != nil {
			return nil, false, err
		}
		items = append(items, page...)

		if next == "" {
			return items, false, nil
		}
		if limit > 0 && len(items) >= limit {
			return items, true, nil
		}
		cont = next
	}
}
