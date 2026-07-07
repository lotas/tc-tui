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
