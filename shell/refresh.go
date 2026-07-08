package shell

import "time"

// isTopView reports whether view is currently the topmost view on the
// stack.
func (s *Shell) isTopView(view View) bool {
	top, ok := s.stack.Top()
	return ok && top == view
}

// Invalidate re-fetches the given view if it is currently the topmost view,
// and redraws. Called by the refresh ticker today; a future push-event
// listener (e.g. Pulse) could call this directly instead of on a timer,
// without any view-layer changes.
func (s *Shell) Invalidate(view View) {
	if !s.isTopView(view) {
		return
	}

	res, ok := s.registry.Resolve(view.ResourceName)
	if !ok {
		return
	}

	switch view.Kind {
	case ListKind:
		s.loadList(res, view.Scope, s.currentFacetValue, false)
	case DetailKind:
		s.loadDetail(res, view.SelectedID, false)
	}
}

// startRefreshLoop stops any existing ticker and, if interval > 0, starts a
// new one that invalidates view for as long as it stays topmost on the
// stack.
func (s *Shell) startRefreshLoop(view View, interval time.Duration) {
	s.stopRefreshLoop()

	if interval <= 0 {
		return
	}

	stop := make(chan struct{})
	s.stopRefresh = stop

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.app.QueueUpdateDraw(func() {
					s.Invalidate(view)
				})
			case <-stop:
				return
			}
		}
	}()
}

func (s *Shell) stopRefreshLoop() {
	if s.stopRefresh != nil {
		close(s.stopRefresh)
		s.stopRefresh = nil
	}
}
