package resource

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	tcclient "github.com/taskcluster/taskcluster/v101/clients/client-go"
	"github.com/taskcluster/taskcluster/v101/clients/client-go/tcsecrets"
)

func TestSecretsResourceList(t *testing.T) {
	fake := &fakeTaskcluster{secrets: []string{"proj/foo", "proj/bar"}}
	res := NewSecretsResource(fake)

	rows, err := res.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].ID != "proj/foo" || rows[0].Cells[0] != "proj/foo" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
}

func TestSecretsResourceListError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeTaskcluster{secretsErr: wantErr}
	res := NewSecretsResource(fake)

	_, err := res.List()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestSecretsResourceDescribe(t *testing.T) {
	raw, _ := json.Marshal(map[string]string{"token": "s3cr3t"})
	fake := &fakeTaskcluster{
		secret: &tcsecrets.Secret{
			Expires: tcclient.Time(time.Now()),
			Secret:  raw,
		},
	}
	res := NewSecretsResource(fake)

	detail, err := res.Describe("proj/foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Title != "Secret :: proj/foo" {
		t.Fatalf("unexpected title: %s", detail.Title)
	}
	if !strings.Contains(detail.Body, "s3cr3t") {
		t.Fatalf("expected the secret value to be rendered directly, got body: %s", detail.Body)
	}
}

func TestSecretsResourceDescribeError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeTaskcluster{secretErr: wantErr}
	res := NewSecretsResource(fake)

	_, err := res.Describe("proj/foo")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestSecretsResourceWebURLs(t *testing.T) {
	res := NewSecretsResource(&fakeTaskcluster{})

	if got := res.ListWebURL("https://tc.example.com", ""); got != "https://tc.example.com/secrets" {
		t.Fatalf("unexpected list web url: %s", got)
	}
	if got := res.DetailWebURL("https://tc.example.com", "proj/foo"); got != "https://tc.example.com/secrets/proj%2Ffoo" {
		t.Fatalf("unexpected detail web url: %s", got)
	}
}
