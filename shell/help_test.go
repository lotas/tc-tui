package shell

import (
	"strings"
	"testing"
	"time"

	"github.com/taskcluster/tc-tui/resource"
)

type fakeResource struct {
	name        string
	aliases     []string
	description string
	columns     []resource.Column
}

func (f fakeResource) Name() string                                { return f.name }
func (f fakeResource) Aliases() []string                           { return f.aliases }
func (f fakeResource) Description() string                         { return f.description }
func (f fakeResource) Columns() []resource.Column                  { return f.columns }
func (f fakeResource) List() ([]resource.Row, error)               { return nil, nil }
func (f fakeResource) Describe(id string) (resource.Detail, error) { return resource.Detail{}, nil }
func (f fakeResource) RefreshInterval() time.Duration              { return 0 }

type fakeScopedResource struct {
	fakeResource
	emptyScope string
}

func (f fakeScopedResource) ScopedList(scope string) ([]resource.Row, error) { return nil, nil }
func (f fakeScopedResource) EmptyScopeResource() string                      { return f.emptyScope }

func TestBuildHelpTextListsGlobalKeys(t *testing.T) {
	text := buildHelpText(resource.NewRegistry())

	for _, want := range []string{"q", ":", "/", "Esc", "?"} {
		if !strings.Contains(text, want) {
			t.Errorf("buildHelpText() missing global key %q\ngot:\n%s", want, text)
		}
	}
}

func TestBuildHelpTextExplainsSorting(t *testing.T) {
	text := buildHelpText(resource.NewRegistry())

	for _, want := range []string{"1-9", "sort", "reverse"} {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(want)) {
			t.Errorf("buildHelpText() missing sorting explanation containing %q\ngot:\n%s", want, text)
		}
	}
}

func TestBuildHelpTextListsPlainResource(t *testing.T) {
	registry := resource.NewRegistry()
	registry.Register(fakeResource{
		name:        "roles",
		aliases:     []string{"role"},
		description: "IAM-style roles and the scopes they grant",
		columns:     []resource.Column{{Title: "ROLE ID"}},
	})

	text := buildHelpText(registry)

	for _, want := range []string{"roles", "role", "IAM-style roles and the scopes they grant", "ROLE ID"} {
		if !strings.Contains(text, want) {
			t.Errorf("buildHelpText() missing %q\ngot:\n%s", want, text)
		}
	}
}

func TestBuildHelpTextFlagsScopedResource(t *testing.T) {
	registry := resource.NewRegistry()
	registry.Register(fakeScopedResource{
		fakeResource: fakeResource{
			name:        "workers",
			aliases:     []string{"w"},
			description: "Individual workers within a worker pool (scoped list)",
			columns:     []resource.Column{{Title: "STATE"}},
		},
		emptyScope: "workerpools",
	})

	text := buildHelpText(registry)

	for _, want := range []string{"workers", "requires a scope", "workerpools"} {
		if !strings.Contains(text, want) {
			t.Errorf("buildHelpText() missing %q\ngot:\n%s", want, text)
		}
	}
}
