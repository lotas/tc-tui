package resource

import (
	"testing"
	"time"
)

type stubResource struct {
	name    string
	aliases []string
}

func (s stubResource) Name() string                       { return s.name }
func (s stubResource) Aliases() []string                  { return s.aliases }
func (s stubResource) Description() string                { return "" }
func (s stubResource) Columns() []Column                  { return nil }
func (s stubResource) List() ([]Row, error)               { return nil, nil }
func (s stubResource) Describe(id string) (Detail, error) { return Detail{}, nil }
func (s stubResource) RefreshInterval() time.Duration     { return 0 }

func TestRegistryResolveByName(t *testing.T) {
	r := NewRegistry()
	r.Register(stubResource{name: "roles", aliases: []string{"role"}})

	res, ok := r.Resolve("roles")
	if !ok || res.Name() != "roles" {
		t.Fatalf("expected to resolve 'roles', got %v, %v", res, ok)
	}
}

func TestRegistryResolveByAlias(t *testing.T) {
	r := NewRegistry()
	r.Register(stubResource{name: "workerpools", aliases: []string{"wp", "pools"}})

	res, ok := r.Resolve("wp")
	if !ok || res.Name() != "workerpools" {
		t.Fatalf("expected to resolve alias 'wp' to 'workerpools', got %v, %v", res, ok)
	}
}

func TestRegistryResolveCaseInsensitive(t *testing.T) {
	r := NewRegistry()
	r.Register(stubResource{name: "roles"})

	if _, ok := r.Resolve("ROLES"); !ok {
		t.Fatalf("expected case-insensitive resolution to succeed")
	}
}

func TestRegistryResolveUnknown(t *testing.T) {
	r := NewRegistry()
	r.Register(stubResource{name: "roles"})

	if _, ok := r.Resolve("nope"); ok {
		t.Fatalf("expected unknown resource to fail resolution")
	}
}

func TestRegistryNamesReturnsRegistrationOrder(t *testing.T) {
	r := NewRegistry()
	r.Register(stubResource{name: "workerpools"})
	r.Register(stubResource{name: "roles"})

	names := r.Names()
	if len(names) != 2 || names[0] != "workerpools" || names[1] != "roles" {
		t.Fatalf("unexpected names: %v", names)
	}
}
