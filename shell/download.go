package shell

import (
	"fmt"
	"os"

	"github.com/taskcluster/tc-tui/resource"
)

// currentDownloadableTarget resolves which resource+id a save-to-disk action
// should operate on: the entity a Detail view is showing, or whichever row
// is currently highlighted on a List view — the latter lets 's' save e.g. an
// artifact straight from the artifacts list without first "stepping into"
// its own Detail page.
func (s *Shell) currentDownloadableTarget() (resourceName, id string, ok bool) {
	top, hasTop := s.stack.Top()
	if !hasTop {
		return "", "", false
	}

	if top.Kind == DetailKind {
		return top.ResourceName, top.SelectedID, true
	}

	row, ok := s.table.SelectedRow()
	if !ok {
		return "", "", false
	}
	return top.ResourceName, row.ID, true
}

// currentDownloadable resolves the Downloadable resource+id+suggested
// filename for whatever entity currentDownloadableTarget points at,
// mirroring currentWebURL's shape for the analogous 'o' key.
func (s *Shell) currentDownloadable() (d resource.Downloadable, id, filename string, ok bool) {
	resourceName, entityID, ok := s.currentDownloadableTarget()
	if !ok {
		return nil, "", "", false
	}

	res, ok := s.registry.Resolve(resourceName)
	if !ok {
		return nil, "", "", false
	}

	downloadable, ok := res.(resource.Downloadable)
	if !ok {
		return nil, "", "", false
	}

	filename, ok = downloadable.DownloadFilename(entityID)
	if !ok {
		return nil, "", "", false
	}

	return downloadable, entityID, filename, true
}

// promptSaveToDisk opens a footer prompt (the 's' key) pre-filled with the
// current Detail view's suggested filename. A bare filename with no
// directory component saves into the process's current working directory,
// matching os.WriteFile's own relative-path behavior.
func (s *Shell) promptSaveToDisk() {
	d, id, filename, ok := s.currentDownloadable()
	if !ok {
		s.showTransientWarning("nothing to save for this view")
		return
	}

	s.openIDPrompt("save as", func(path string) {
		s.saveToDisk(d, id, path)
	})
	s.footerInput.SetText(filename)
}

// saveToDisk fetches id's raw content and writes it to path, refusing to
// overwrite a file that's already there — the user has to type a different
// name instead of the save silently clobbering something.
func (s *Shell) saveToDisk(d resource.Downloadable, id, path string) {
	if path == "" {
		return
	}

	if _, err := os.Stat(path); err == nil {
		s.showTransientWarning(fmt.Sprintf("%s already exists — choose a different name", path))
		return
	}

	content, truncated, err := d.DownloadContent(id)
	if err != nil {
		s.showTransientWarning(fmt.Sprintf("failed to fetch content: %s", err))
		return
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		s.showTransientWarning(fmt.Sprintf("failed to save %s: %s", path, err))
		return
	}

	msg := fmt.Sprintf("saved to %s", path)
	if truncated {
		msg += " (truncated — content exceeded the fetch size cap)"
	}
	s.showTransientInfo(msg)
}
