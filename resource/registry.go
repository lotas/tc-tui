package resource

import "strings"

// Registry maps resource names and aliases to Resource implementations. It
// is built once at startup and never mutated afterwards.
type Registry struct {
	byName  map[string]Resource
	aliases map[string]string
	order   []string
}

func NewRegistry() *Registry {
	return &Registry{
		byName:  make(map[string]Resource),
		aliases: make(map[string]string),
	}
}

func (r *Registry) Register(res Resource) {
	name := strings.ToLower(res.Name())

	r.byName[name] = res
	r.order = append(r.order, res.Name())

	for _, alias := range res.Aliases() {
		r.aliases[strings.ToLower(alias)] = name
	}
}

// Resolve looks up a resource by name or alias, case-insensitively. If a
// name and an alias collide, the name match always takes priority.
func (r *Registry) Resolve(nameOrAlias string) (Resource, bool) {
	key := strings.ToLower(nameOrAlias)

	if res, ok := r.byName[key]; ok {
		return res, true
	}

	if name, ok := r.aliases[key]; ok {
		return r.byName[name], true
	}

	return nil, false
}

// Names returns registered resource names in registration order.
func (r *Registry) Names() []string {
	return append([]string(nil), r.order...)
}
