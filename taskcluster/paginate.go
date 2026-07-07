package taskcluster

// paginate repeatedly calls fetch with the continuation token from the
// previous call, collecting all pages, until fetch returns an empty token.
func paginate[T any](fetch func(continuationToken string) ([]T, string, error)) ([]T, error) {
	items := make([]T, 0)
	cont := ""

	for {
		page, next, err := fetch(cont)
		if err != nil {
			return nil, err
		}
		items = append(items, page...)

		if next == "" {
			break
		}
		cont = next
	}

	return items, nil
}
