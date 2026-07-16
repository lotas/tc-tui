package taskcluster

import (
	"errors"
	"testing"
)

func TestPaginateCollectsAllPages(t *testing.T) {
	pages := [][]int{{1, 2}, {3, 4}, {5}}
	tokens := []string{"page2", "page3", ""}
	calls := 0
	var receivedConts []string

	result, err := paginate(func(cont string) ([]int, string, error) {
		receivedConts = append(receivedConts, cont)
		page := pages[calls]
		token := tokens[calls]
		calls++
		return page, token, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}

	expectedConts := []string{"", "page2", "page3"}
	if len(receivedConts) != len(expectedConts) {
		t.Fatalf("expected conts %v, got %v", expectedConts, receivedConts)
	}
	for i, c := range expectedConts {
		if receivedConts[i] != c {
			t.Fatalf("expected conts %v, got %v", expectedConts, receivedConts)
		}
	}

	expected := []int{1, 2, 3, 4, 5}
	if len(result) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
	for i, v := range expected {
		if result[i] != v {
			t.Fatalf("expected %v, got %v", expected, result)
		}
	}
}

func TestPaginateStopsOnFirstError(t *testing.T) {
	wantErr := errors.New("boom")
	calls := 0

	_, err := paginate(func(cont string) ([]int, string, error) {
		calls++
		return nil, "", wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestPaginateUpToStopsAtLimitAndReportsTruncated(t *testing.T) {
	pages := [][]int{{1, 2}, {3, 4}, {5, 6}}
	tokens := []string{"page2", "page3", ""}
	calls := 0

	result, truncated, err := paginateUpTo(3, func(cont string) ([]int, string, error) {
		page := pages[calls]
		token := tokens[calls]
		calls++
		return page, token, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected fetching to stop after 2 pages (4 items >= limit 3), got %d calls", calls)
	}
	if !truncated {
		t.Fatalf("expected truncated=true with a continuation token still remaining")
	}
	if len(result) != 4 {
		t.Fatalf("expected the already-fetched items to be kept (4), got %v", result)
	}
}

func TestPaginateUpToNotTruncatedWhenPagesRunOutBeforeLimit(t *testing.T) {
	pages := [][]int{{1, 2}, {3}}
	tokens := []string{"page2", ""}
	calls := 0

	result, truncated, err := paginateUpTo(100, func(cont string) ([]int, string, error) {
		page := pages[calls]
		token := tokens[calls]
		calls++
		return page, token, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if truncated {
		t.Fatalf("expected truncated=false when every page was fetched")
	}
	if len(result) != 3 {
		t.Fatalf("expected all 3 items, got %v", result)
	}
}

func TestPaginateUpToNotTruncatedWhenLimitHitOnFinalPage(t *testing.T) {
	// Exactly `limit` items arrive AND the token runs out on the same page —
	// nothing was left unfetched, so this must not read as truncated.
	result, truncated, err := paginateUpTo(2, func(cont string) ([]int, string, error) {
		return []int{1, 2}, "", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if truncated {
		t.Fatalf("expected truncated=false when the final page lands exactly on the limit")
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %v", result)
	}
}

func TestPaginateUpToZeroLimitFetchesAllPages(t *testing.T) {
	pages := [][]int{{1, 2}, {3, 4}, {5}}
	tokens := []string{"page2", "page3", ""}
	calls := 0

	result, truncated, err := paginateUpTo(0, func(cont string) ([]int, string, error) {
		page := pages[calls]
		token := tokens[calls]
		calls++
		return page, token, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected limit 0 to fetch every page, got %d calls", calls)
	}
	if truncated {
		t.Fatalf("expected truncated=false with limit 0")
	}
	if len(result) != 5 {
		t.Fatalf("expected all 5 items, got %v", result)
	}
}

func TestPaginateUpToStopsOnFirstError(t *testing.T) {
	wantErr := errors.New("boom")
	calls := 0

	_, _, err := paginateUpTo(10, func(cont string) ([]int, string, error) {
		calls++
		return nil, "", wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestPaginateSinglePage(t *testing.T) {
	calls := 0

	result, err := paginate(func(cont string) ([]string, string, error) {
		calls++
		return []string{"a", "b"}, "", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
	if len(result) != 2 || result[0] != "a" || result[1] != "b" {
		t.Fatalf("unexpected result: %v", result)
	}
}
