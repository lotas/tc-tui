package resource

import (
	"fmt"
	"strings"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

// maxHookFiresShown caps how many recent fires a hook's Detail view renders —
// the hooks service retains a rolling window of past fires, and a frequently
// triggered hook's full history would bury the definition below it.
const maxHookFiresShown = 20

type HooksResource struct {
	tc taskcluster.Taskcluster
}

func NewHooksResource(tc taskcluster.Taskcluster) *HooksResource {
	return &HooksResource{tc: tc}
}

func (r *HooksResource) Name() string      { return "hooks" }
func (r *HooksResource) Aliases() []string { return []string{"hook"} }
func (r *HooksResource) Description() string {
	return "Hooks (scheduled/triggered task templates) across all hook groups"
}

func (r *HooksResource) Columns() []Column {
	return []Column{
		{Title: "HOOK", Expand: true},
		{Title: "NAME", Width: 30},
		{Title: "SCHEDULE", Width: 30},
	}
}

func (r *HooksResource) List() ([]Row, error) {
	hooks, err := r.tc.GetHooks()
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(hooks))
	for _, h := range hooks {
		id := h.HookGroupID + "/" + h.HookID
		rows = append(rows, Row{
			ID:    id,
			Cells: []string{id, h.Metadata.Name, strings.Join(h.Schedule, ", ")},
		})
	}

	return rows, nil
}

// splitHookID splits a row ID back into its hookGroupId/hookId pair. A hook
// group ID can never contain "/" (its schema forbids it) while a hook ID may,
// so splitting on the first "/" is unambiguous.
func splitHookID(id string) (group, hook string, err error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid hook id %q, expected hookGroupId/hookId", id)
	}
	return parts[0], parts[1], nil
}

func (r *HooksResource) Describe(id string) (Detail, error) {
	group, hookID, err := splitHookID(id)
	if err != nil {
		return Detail{}, err
	}

	h, err := r.tc.GetHook(group, hookID)
	if err != nil {
		return Detail{}, err
	}

	schedule := "(none)"
	if len(h.Schedule) > 0 {
		schedule = strings.Join(h.Schedule, "\n")
	}

	fires, firesErr := r.tc.GetHookLastFires(group, hookID)
	sortHookFiresNewestFirst(fires)

	body := fmt.Sprintf(
		"%s%s"+
			"[green]Description:[white]\n%s\n\n"+
			"[green]Schedule:[white]\n%s\n\n"+
			"[green]Recent Fires (%d):[white]\n%s\n"+
			"[green]Task Template:[white]\n%s\n\n"+
			"[green]Trigger Schema:[white]\n%s",
		fieldRow(40, "Name", h.Metadata.Name, "Owner", h.Metadata.Owner),
		fieldRow(40, "Email on Error", fmt.Sprint(h.Metadata.EmailOnError)),
		renderMarkdown(h.Metadata.Description),
		schedule,
		len(fires), renderHookFires(fires, firesErr),
		renderYAML(h.Task),
		renderYAML(h.TriggerSchema),
	)

	detail := Detail{Title: fmt.Sprintf("Hook :: %s", id), Body: body}

	// Every fire is one 'f' away as a navigable list; the most recent
	// fire's task additionally gets its own one-key jump.
	detail.Actions = append(detail.Actions, DetailAction{
		Key:    'f',
		Label:  "fires",
		Target: NavTarget{ResourceName: "hookfires", ID: id, Kind: NavScopedList},
	})
	for _, f := range fires {
		if f.TaskID != "" {
			detail.Actions = append(detail.Actions, DetailAction{
				Key:    't',
				Label:  "last task",
				Target: NavTarget{ResourceName: "task", ID: f.TaskID, Kind: NavDetail},
			})
			break
		}
	}

	return detail, nil
}

// renderHookFires renders a hook's recent fires, newest first, one per line.
// Fires are best-effort enrichment of the Detail view: a listing failure
// (err non-nil) renders as an inline notice rather than failing the whole
// Describe — the hook's definition is still worth showing.
func renderHookFires(fires taskcluster.HookLastFireList, err error) string {
	if err != nil {
		return fmt.Sprintf("[red]unavailable:[white] %v\n", err)
	}
	if len(fires) == 0 {
		return "(none)\n"
	}

	var b strings.Builder
	for i, f := range fires {
		if i == maxHookFiresShown {
			fmt.Fprintf(&b, "... and %d more\n", len(fires)-maxHookFiresShown)
			break
		}

		fmt.Fprintf(&b, "%s  %s  %s  %s  %s\n",
			renderHookFireResult(fmt.Sprintf("%-7s", f.Result)),
			fmt.Sprint(f.TaskCreateTime),
			f.TaskID,
			renderTaskState(f.TaskState),
			f.FiredBy,
		)
		if f.Error != "" {
			fmt.Fprintf(&b, "         [red]%s[white]\n", firstLine(f.Error, 160))
		}
	}
	return b.String()
}

// firstLine returns s's first line, truncated to at most max runes — used
// for error blobs that can run to a full stack trace where a single
// summary line is enough context.
func firstLine(s string, max int) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if runes := []rune(s); len(runes) > max {
		return string(runes[:max]) + "..."
	}
	return s
}

// RefreshInterval is longer than most other list resources' (15s) — one
// list refresh costs 1+N API calls (listHookGroups, then listHooks per
// group), and hook definitions change rarely.
func (r *HooksResource) RefreshInterval() time.Duration { return 60 * time.Second }

func (r *HooksResource) ListWebURL(rootURL, scope string) string {
	return webUIPath(rootURL, "hooks")
}

func (r *HooksResource) DetailWebURL(rootURL, id string) string {
	return hookWebURL(rootURL, id)
}
