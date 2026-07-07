package shell

type ViewKind int

const (
	ListKind ViewKind = iota
	DetailKind
)

// View is one entry in the navigation stack: which resource, whether it's
// the list or a single entity's detail, and (for detail) which entity.
type View struct {
	ResourceName string
	Kind         ViewKind
	SelectedID   string
}

// ViewStack replaces the old single lastPage field with real multi-level
// breadcrumbs. Esc pops one level; selecting a row or running a `:` command
// pushes.
type ViewStack struct {
	views []View
}

func NewViewStack() *ViewStack {
	return &ViewStack{}
}

func (s *ViewStack) Push(v View) {
	s.views = append(s.views, v)
}

func (s *ViewStack) Pop() (View, bool) {
	if len(s.views) == 0 {
		return View{}, false
	}

	v := s.views[len(s.views)-1]
	s.views = s.views[:len(s.views)-1]

	return v, true
}

func (s *ViewStack) Top() (View, bool) {
	if len(s.views) == 0 {
		return View{}, false
	}

	return s.views[len(s.views)-1], true
}

func (s *ViewStack) Len() int {
	return len(s.views)
}

// ResetTo replaces the entire stack with a single root view. Used when the
// `:` command bar switches resource type — it doesn't nest under the
// current view.
func (s *ViewStack) ResetTo(v View) {
	s.views = []View{v}
}
