package shell

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/taskcluster/tc-tui/resource"
)

// actionModalWidth/Height size the centered action dialog. A multiline
// (YAML/JSON/text) action gets a taller, wider box for its text area; a
// confirm-only or single-line action gets a compact one.
const (
	actionModalWidth          = 78
	actionModalWidthMultiline = 96
	actionModalHeight         = 11
	actionModalHeightMulti    = 26
	actionTextAreaHeight      = 12
)

// ActionView renders the shared authenticated-action dialog: a centered,
// bordered box holding a warning/prompt message, an optional input field
// (single-line or a multi-line text area, per the Action's InputMode), an
// optional "type the confirm word" field for a destructive action, Confirm/
// Cancel buttons, and a status line for validation errors, progress, and API
// failures. It is generic over resource.Action — it knows nothing about what
// any particular action does; the Shell wires the submit/cancel callbacks and
// drives validation, Perform, and cache invalidation around it.
type ActionView struct {
	*tview.Flex // the centered wrapper; the primitive added to s.content

	message *tview.TextView
	form    *tview.Form
	status  *tview.TextView

	// inputIndex/confirmIndex are the form-item positions of the value input
	// and the destructive confirm-word field, or -1 when the current action
	// has no such field. Buttons are not form items, so these index only the
	// text inputs, in the order they were added.
	inputIndex   int
	confirmIndex int

	onSubmit func()
	onCancel func()
}

func NewActionView() *ActionView {
	v := &ActionView{
		Flex:         tview.NewFlex(),
		message:      tview.NewTextView().SetDynamicColors(true).SetWordWrap(true),
		status:       tview.NewTextView().SetDynamicColors(true).SetWordWrap(true),
		inputIndex:   -1,
		confirmIndex: -1,
	}
	return v
}

// SetAction rebuilds the dialog for a. onSubmit is invoked when the user
// activates the Confirm button (the Shell then validates and performs);
// onCancel when they pick Cancel or press Esc.
func (v *ActionView) SetAction(a resource.Action, onSubmit, onCancel func()) {
	v.onSubmit = onSubmit
	v.onCancel = onCancel
	v.inputIndex = -1
	v.confirmIndex = -1

	v.message.SetText(actionMessage(a))
	v.status.SetText("")

	form := tview.NewForm()
	form.SetButtonsAlign(tview.AlignRight)

	next := 0
	if a.Input != resource.InputNone {
		label := a.InputLabel
		if label == "" {
			label = "value"
		}
		if a.Input.Multiline() {
			form.AddTextArea(label, a.InitialText, 0, actionTextAreaHeight, 0, nil)
		} else {
			form.AddInputField(label, a.InitialText, 0, nil, nil)
		}
		v.inputIndex = next
		next++
	}
	if a.Destructive && a.ConfirmWord != "" {
		form.AddInputField(fmt.Sprintf("type %q to confirm", a.ConfirmWord), "", 0, nil, nil)
		v.confirmIndex = next
		next++
	}

	form.AddButton("Confirm", func() {
		if v.onSubmit != nil {
			v.onSubmit()
		}
	})
	form.AddButton("Cancel", func() {
		if v.onCancel != nil {
			v.onCancel()
		}
	})
	form.SetCancelFunc(func() {
		if v.onCancel != nil {
			v.onCancel()
		}
	})
	// A confirm-only action has no fields to fill in — start focus on the
	// Confirm button so Enter alone can drive it. An action with input starts
	// on the first field instead.
	if a.Input == resource.InputNone {
		form.SetFocus(form.GetFormItemCount())
	}
	v.form = form

	box := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.message, actionMessageHeight(a), 0, false).
		AddItem(form, 0, 1, true).
		AddItem(v.status, 2, 0, false)
	box.SetBorder(true).SetTitle(actionTitle(a))
	if a.Destructive {
		box.SetBorderColor(tcell.ColorRed).SetTitleColor(tcell.ColorRed)
	}

	width, height := actionModalWidth, actionModalHeight
	if a.Input.Multiline() {
		width, height = actionModalWidthMultiline, actionModalHeightMulti
	}

	// Rebuild the centered wrapper (spacer | box | spacer, vertically and
	// horizontally) so the box floats mid-screen at the size this action
	// needs. Clear() leaves the root Flex reusable — s.content keeps
	// referencing the same primitive across successive actions.
	v.Clear()
	v.SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(box, height, 0, true).
			AddItem(nil, 0, 1, false), width, 0, true).
		AddItem(nil, 0, 1, false)
}

// InputText returns the current value-input text, or "" when the action has
// no input field.
func (v *ActionView) InputText() string {
	return v.formItemText(v.inputIndex)
}

// ConfirmWordText returns what the user typed into the destructive confirm-
// word field, or "" when the action has none.
func (v *ActionView) ConfirmWordText() string {
	return v.formItemText(v.confirmIndex)
}

func (v *ActionView) formItemText(index int) string {
	if index < 0 || v.form == nil {
		return ""
	}
	switch item := v.form.GetFormItem(index).(type) {
	case *tview.TextArea:
		return item.GetText()
	case *tview.InputField:
		return item.GetText()
	default:
		return ""
	}
}

// setFieldText writes text into the given form field — used only by tests to
// simulate typing, since a SimulationScreen can't send real keystrokes into a
// focused field ergonomically.
func (v *ActionView) setFieldText(index int, text string) {
	if index < 0 || v.form == nil {
		return
	}
	switch item := v.form.GetFormItem(index).(type) {
	case *tview.TextArea:
		item.SetText(text, true)
	case *tview.InputField:
		item.SetText(text)
	}
}

// SetStatus shows a status line under the form — a validation message or an
// API failure (isError, red) or neutral progress (not isError, yellow).
func (v *ActionView) SetStatus(msg string, isError bool) {
	color := "yellow"
	if isError {
		color = "red"
	}
	v.status.SetText(fmt.Sprintf("[%s]%s[white]", color, tview.Escape(msg)))
}

// actionTitle is the dialog's bordered title.
func actionTitle(a resource.Action) string {
	if a.Destructive {
		return fmt.Sprintf(" [!] %s ", a.Label)
	}
	return fmt.Sprintf(" %s ", a.Label)
}

// actionMessage is the warning/prompt block shown above the form: a stern
// destructive banner (when applicable) followed by the action's prompt.
func actionMessage(a resource.Action) string {
	prompt := tview.Escape(a.Prompt)
	if a.Destructive {
		return "[red]⚠ This action is destructive and cannot be undone.[white]\n\n[white]" + prompt
	}
	return "[white]" + prompt
}

// actionMessageHeight budgets rows for actionMessage — enough for the
// two-line destructive banner plus a wrapped prompt, or just a wrapped
// prompt otherwise.
func actionMessageHeight(a resource.Action) int {
	if a.Destructive {
		return 5
	}
	return 3
}

// --- Shell integration -----------------------------------------------------

// startAction opens the action dialog for a, remembering which content page
// to restore on close. The Shell drives the rest of the flow (submitAction,
// finishAction, closeAction).
func (s *Shell) startAction(a resource.Action) {
	s.actionReturnPage, _ = s.content.GetFrontPage()
	s.actionOpen = true
	s.actionBusy = false
	s.currentAction = a

	s.actionView.SetAction(a,
		func() { s.submitAction() },
		func() { s.closeAction() },
	)

	s.content.SwitchToPage(pageAction)
	s.updateBorderColor()
	s.app.SetFocus(s.actionView.form)
}

// closeAction dismisses the dialog and returns to the page it was launched
// from, without performing anything.
func (s *Shell) closeAction() {
	if !s.actionOpen {
		return
	}
	s.actionOpen = false
	s.actionBusy = false

	page := s.actionReturnPage
	if page == "" || page == pageAction {
		page = pageDetail
	}
	s.content.SwitchToPage(page)
	s.updateBorderColor()
	s.app.SetFocus(s.activeContent)
}

// submitAction validates the collected input against the current action and,
// if it passes, runs Perform off the UI thread with a progress indicator. A
// validation failure stays in the dialog with a red status so the user can
// fix it; an API failure does the same after Perform returns.
func (s *Shell) submitAction() {
	if s.actionBusy {
		return // Perform already in flight; ignore a double activation
	}
	a := s.currentAction

	if a.Destructive && a.ConfirmWord != "" {
		if s.actionView.ConfirmWordText() != a.ConfirmWord {
			s.actionView.SetStatus(fmt.Sprintf("type %q exactly to confirm", a.ConfirmWord), true)
			return
		}
	}

	input, err := resource.ParseActionInput(a.Input, s.actionView.InputText(), !a.OptionalInput)
	if err != nil {
		s.actionView.SetStatus(err.Error(), true)
		return
	}
	if a.Validate != nil {
		if err := a.Validate(input); err != nil {
			s.actionView.SetStatus(err.Error(), true)
			return
		}
	}

	s.actionBusy = true
	s.actionView.SetStatus("Working…", false)

	go func() {
		err := a.Perform(input)
		s.app.QueueUpdateDraw(func() {
			s.actionBusy = false
			if err != nil {
				s.actionView.SetStatus(fmt.Sprintf("failed: %s", err), true)
				return
			}
			s.finishAction(a)
		})
	}()
}

// finishAction runs once Perform succeeds: it drops the caches the mutation
// invalidated (plus the current list's own, so the change is picked up on the
// next load), closes the dialog, and reports success in the footer.
//
// For a non-destructive edit it also force-refreshes the view it was launched
// from, so the new state is visible immediately — the fresh content is the
// real confirmation. A destructive action deliberately does NOT auto-refresh:
// the entity may no longer exist (a re-Describe would 404 into an error
// screen), and skipping the refresh also lets the "done" toast survive rather
// than being wiped by the re-render. The invalidated cache still guarantees
// the affected lists show the removal on their next load or auto-refresh
// tick.
func (s *Shell) finishAction(a resource.Action) {
	for _, name := range a.Invalidates {
		s.cache.invalidate(name)
	}
	if top, ok := s.stack.Top(); ok && top.Kind == ListKind {
		s.cache.invalidate(top.ResourceName)
	}

	s.closeAction()

	label := a.Label
	if label == "" {
		label = "action"
	}
	s.showTransientInfo(fmt.Sprintf("%s: done", label))

	if !a.Destructive {
		// refreshCurrent re-fetches the top view; the list cache was just
		// dropped above so a list re-fetches fresh, and a detail always
		// re-Describes. Its re-render replaces the toast above, which is fine
		// here — the refreshed content is the confirmation.
		s.refreshCurrent()
	}
}

// currentActionTarget resolves which Actionable resource + entity id the
// action keys apply to: the entity a Detail view shows, or the row currently
// highlighted on a List view (so an action can fire straight from a list
// without stepping into the entity first). Mirrors currentDownloadableTarget.
func (s *Shell) currentActionTarget() (res resource.Actionable, id string, ok bool) {
	top, hasTop := s.stack.Top()
	if !hasTop {
		return nil, "", false
	}

	var resourceName, entityID string
	if top.Kind == DetailKind {
		resourceName, entityID = top.ResourceName, top.SelectedID
	} else {
		row, rok := s.table.SelectedRow()
		if !rok {
			return nil, "", false
		}
		resourceName, entityID = top.ResourceName, row.ID
	}

	r, rok := s.registry.Resolve(resourceName)
	if !rok {
		return nil, "", false
	}
	act, aok := r.(resource.Actionable)
	if !aok {
		return nil, "", false
	}
	return act, entityID, true
}

// resolveActionByKey finds the action bound to key on the current target, if
// any — the dispatch used by globalInputCapture.
func (s *Shell) resolveActionByKey(key rune) (resource.Action, bool) {
	act, id, ok := s.currentActionTarget()
	if !ok {
		return resource.Action{}, false
	}
	for _, a := range act.Actions(id) {
		if a.Key == key {
			return a, true
		}
	}
	return resource.Action{}, false
}
