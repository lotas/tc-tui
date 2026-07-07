package shell

import "testing"

func TestViewStackPushPopTop(t *testing.T) {
	s := NewViewStack()

	if _, ok := s.Pop(); ok {
		t.Fatalf("expected pop on empty stack to fail")
	}
	if _, ok := s.Top(); ok {
		t.Fatalf("expected top on empty stack to fail")
	}

	s.Push(View{ResourceName: "workerpools", Kind: ListKind})
	s.Push(View{ResourceName: "workerpools", Kind: DetailKind, SelectedID: "pool-a"})

	if s.Len() != 2 {
		t.Fatalf("expected len 2, got %d", s.Len())
	}

	top, ok := s.Top()
	if !ok || top.SelectedID != "pool-a" {
		t.Fatalf("unexpected top: %+v, %v", top, ok)
	}

	popped, ok := s.Pop()
	if !ok || popped.SelectedID != "pool-a" {
		t.Fatalf("unexpected pop: %+v, %v", popped, ok)
	}
	if s.Len() != 1 {
		t.Fatalf("expected len 1 after pop, got %d", s.Len())
	}

	top, ok = s.Top()
	if !ok || top.Kind != ListKind || top.SelectedID != "" {
		t.Fatalf("unexpected top after pop: %+v, %v", top, ok)
	}
}

func TestViewStackResetTo(t *testing.T) {
	s := NewViewStack()
	s.Push(View{ResourceName: "workerpools", Kind: ListKind})
	s.Push(View{ResourceName: "workerpools", Kind: DetailKind, SelectedID: "pool-a"})

	s.ResetTo(View{ResourceName: "roles", Kind: ListKind})

	if s.Len() != 1 {
		t.Fatalf("expected len 1 after ResetTo, got %d", s.Len())
	}
	top, ok := s.Top()
	if !ok || top.ResourceName != "roles" {
		t.Fatalf("unexpected top after ResetTo: %+v, %v", top, ok)
	}
}

func TestViewStackPreservesScope(t *testing.T) {
	s := NewViewStack()
	s.Push(View{ResourceName: "workers", Kind: ListKind, Scope: "gcp/pool-a"})

	top, ok := s.Top()
	if !ok || top.Scope != "gcp/pool-a" {
		t.Fatalf("unexpected top: %+v, %v", top, ok)
	}

	popped, ok := s.Pop()
	if !ok || popped.Scope != "gcp/pool-a" {
		t.Fatalf("unexpected pop: %+v, %v", popped, ok)
	}

	s.Push(View{ResourceName: "workers", Kind: ListKind, Scope: "gcp/pool-a"})
	s.ResetTo(View{ResourceName: "roles", Kind: ListKind})

	top, ok = s.Top()
	if !ok || top.Scope != "" {
		t.Fatalf("expected ResetTo to clear scope, got: %+v, %v", top, ok)
	}
}
