package shell

import "time"

// Invalidate re-fetches the given resource/id if it is currently the
// topmost view, and redraws. Called by the refresh ticker today; a future
// push-event listener (e.g. Pulse) could call this directly instead of on a
// timer, without any view-layer changes.
func (s *Shell) Invalidate(resourceName, id string) {
	top, ok := s.stack.Top()
	if !ok || top.ResourceName != resourceName || top.SelectedID != id {
		return
	}

	res, ok := s.registry.Resolve(resourceName)
	if !ok {
		return
	}

	switch top.Kind {
	case ListKind:
		s.loadList(res, false)
	case DetailKind:
		s.loadDetail(res, id, false)
	}
}

// startRefreshLoop stops any existing ticker and, if interval > 0, starts a
// new one that calls Invalidate for as long as this resource/id stays
// topmost on the stack.
func (s *Shell) startRefreshLoop(resourceName, id string, interval time.Duration) {
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
					s.Invalidate(resourceName, id)
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
