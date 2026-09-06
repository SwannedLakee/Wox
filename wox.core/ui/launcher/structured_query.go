package launcher

import (
	"context"
	"reflect"
	"strings"
	"wox/common"
	woxui "wox/ui/runtime"
	"wox/util"
)

type structureResolver interface {
	ResolveQueryHint(context.Context, string) *common.QueryHint
}

// structuredQueryEditor tracks semantic ranges and document-level undo for one continuous text editor.
type structuredQueryEditor struct {
	active      int
	prefix      string
	candidate   *common.QueryHint
	suppressed  string
	allSelected bool
	undo        []structuredQuerySnapshot
	redo        []structuredQuerySnapshot
}

type structuredQuerySnapshot struct {
	structure *common.QueryHint
	text      string
	active    int
	prefix    string
}

// rememberStructuredQuery snapshots content before an element edit or a structural transition.
func (a *App) rememberStructuredQuery() {
	s := &a.structured
	if len(s.undo) > 0 {
		last := s.undo[len(s.undo)-1]
		if last.text == a.query.QueryText && last.active == s.active && reflect.DeepEqual(last.structure, a.query.QueryHint) {
			s.redo = nil
			return
		}
	}
	s.undo = append(s.undo, structuredQuerySnapshot{a.query.QueryHint.Clone(), a.query.QueryText, s.active, s.prefix})
	if len(s.undo) > 100 {
		s.undo = s.undo[len(s.undo)-100:]
	}
	s.redo = nil
}

// installStructuredQuery installs semantic content without splitting the native text editor.
func (a *App) installStructuredQuery(structure *common.QueryHint) {
	a.query.QueryHint = structure.Clone()
	a.query.QueryText = structure.PlainText()
	a.structured.active = 0
	a.structured.prefix = ""
	a.structured.candidate = nil
	if len(structure.Elements) > 0 && structure.Elements[0].Kind == common.QueryElementText {
		a.structured.prefix = structure.Elements[0].Text
	}
	for i, e := range structure.Elements {
		if e.Kind != common.QueryElementText {
			a.structured.active = i
			break
		}
	}
	a.editor.SetText(structure.PlainText(), false)
	start, end := queryElementRange(structure, a.structured.active)
	a.editor.SetCaret(end)
	if structure.Elements[a.structured.active].Kind == common.QueryElementBlock {
		a.editor.SetSelection(start, end)
	}
}

// updateStructuredText maps continuous edits back to semantic values, or offers a command template.
func (a *App) updateStructuredText(text string) string {
	s := &a.structured
	if structure := a.query.QueryHint; structure != nil {
		a.rememberStructuredQuery()
		// A local edit retains its argument identity. Edits across semantic boundaries
		// keep the user's text but discard metadata that can no longer be trusted.
		old, updated := []rune(structure.PlainText()), []rune(text)
		start := 0
		for start < len(old) && start < len(updated) && old[start] == updated[start] {
			start++
		}
		end, newEnd := len(old), len(updated)
		for end > start && newEnd > start && old[end-1] == updated[newEnd-1] {
			end--
			newEnd--
		}
		if string(old) == text {
			return text
		}
		owner := -1
		for n := 0; n < len(structure.Elements); n++ {
			i := (s.active + n) % len(structure.Elements)
			left, right := queryElementRange(structure, i)
			if structure.Elements[i].Kind == common.QueryElementArgument && start >= left && end <= right {
				owner = i
				break
			}
		}
		if owner >= 0 {
			next := structure.Clone()
			left, right := queryElementRange(structure, owner)
			next.Elements[owner].Value = string(updated[left : right+len(updated)-len(old)])
			a.query.QueryHint = next
			s.active = owner
		} else {
			a.query.QueryHint = nil
			s.suppressed = text
		}
		s.allSelected = false
		return text
	}
	s.candidate = nil
	if len(s.undo)+len(s.redo) > 0 && text != a.query.QueryText {
		a.rememberStructuredQuery()
	}
	if a.query.QueryType != "input" || text == s.suppressed {
		return text
	}
	s.suppressed = ""
	resolver, ok := a.services.(structureResolver)
	if !ok {
		return text
	}
	structure := resolver.ResolveQueryHint(a.lifecycleCtx, text)
	if structure == nil {
		return text
	}
	if !strings.HasSuffix(text, " ") {
		s.candidate = structure
		return text
	}
	a.rememberStructuredQuery()
	a.installStructuredQuery(structure)
	return structure.PlainText()
}

// structuredQueryChanged shares normal query invalidation without treating full text as a slot edit.
func (a *App) structuredQueryChanged() {
	a.beginQueryGenerationLocked()
	a.beginQueryTransitionLocked(false)
	a.completionHint = nil
	a.canRecallHistory = false
	if a.window != nil {
		_ = a.window.Invalidate()
	}
	a.reconcileSelectedPreview()
	if a.services != nil {
		if err := a.sendCurrentQuery(); err != nil {
			util.GetLogger().Error(a.lifecycleCtx, err.Error())
		}
	}
}

// focusQueryElement changes only local editing state; navigation never starts a query.
func (a *App) focusQueryElement(index int) {
	structure := a.query.QueryHint
	if structure == nil || a.editor.State().Composition != "" {
		return
	}
	direction := 1
	if index < a.structured.active {
		direction = -1
	}
	next := (index + len(structure.Elements)) % len(structure.Elements)
	for count := 0; count < len(structure.Elements) && structure.Elements[next].Kind == common.QueryElementText && strings.TrimSpace(structure.Elements[next].Text) == ""; count++ {
		next = (next + direction + len(structure.Elements)) % len(structure.Elements)
	}
	a.structured.active = next
	a.structured.allSelected = false
	start, end := queryElementRange(structure, next)
	a.editor.SetSelection(start, end)
	if a.window != nil {
		_ = a.window.Invalidate()
	}
}

// onStructuredQueryKey handles document operations before ordinary completion and text shortcuts.
func (a *App) onStructuredQueryKey(event woxui.KeyEvent) bool {
	s := &a.structured
	if event.Modifiers.HasPrimary() && (event.Key == woxui.Key("z") || event.Key == woxui.Key("y")) && len(s.undo)+len(s.redo) > 0 {
		from, to := &s.undo, &s.redo
		if event.Key == woxui.Key("y") || event.Modifiers&woxui.KeyModifierShift != 0 {
			from, to = to, from
		}
		if len(*from) == 0 {
			return true
		}
		*to = append(*to, structuredQuerySnapshot{a.query.QueryHint.Clone(), a.query.QueryText, s.active, s.prefix})
		snapshot := (*from)[len(*from)-1]
		*from = (*from)[:len(*from)-1]
		a.query.QueryHint = snapshot.structure.Clone()
		a.query.QueryText = snapshot.text
		s.active, s.prefix, s.allSelected, s.candidate = snapshot.active, snapshot.prefix, false, nil
		text := snapshot.text
		a.editor.SetText(text, false)
		s.suppressed = snapshot.text
		a.structuredQueryChanged()
		return true
	}
	structure := a.query.QueryHint
	if structure == nil {
		if s.candidate != nil && event.Key == woxui.KeyTab && event.Modifiers == 0 {
			a.rememberStructuredQuery()
			a.installStructuredQuery(s.candidate)
			a.structuredQueryChanged()
			return true
		}
		return false
	}
	if event.Modifiers.HasPrimary() && (event.Key == woxui.Key("c") || event.Key == woxui.Key("x") || event.Key == woxui.Key("v")) {
		a.selectTouchedQueryBlocks()
	}
	if event.Key == woxui.KeyTab && (event.Modifiers == 0 || event.Modifiers == woxui.KeyModifierShift) {
		state := a.editor.State()
		// Pointer and arrow navigation may have moved away from the last Tab target.
		left, right := queryElementRange(structure, s.active)
		if state.Selection.Start() < left || state.Selection.End() > right {
			for i := range structure.Elements {
				start, end := queryElementRange(structure, i)
				if state.Selection.Start() >= start && state.Selection.End() <= end {
					s.active = i
					break
				}
			}
		}
		delta := 1
		if event.Modifiers != 0 {
			delta = -1
		}
		a.focusQueryElement(s.active + delta)
		return true
	}
	state := a.editor.State()
	if event.Key == woxui.KeyArrowLeft || event.Key == woxui.KeyArrowRight || event.Key == woxui.KeyHome || event.Key == woxui.KeyEnd {
		s.allSelected = false
	}
	if event.Key == woxui.Key("a") && event.Modifiers.HasPrimary() {
		a.selectEntireQuery()
		if a.window != nil {
			_ = a.window.Invalidate()
		}
		return true
	}
	if s.allSelected && (event.Key == woxui.KeyBackspace || event.Key == woxui.KeyDelete) {
		a.replaceWholeStructuredQuery("")
		return true
	}
	// Blocks retain atomic deletion; arguments use the ordinary editor's word and
	// character deletion, including selections crossing the command prefix.
	if event.Key == woxui.KeyBackspace || event.Key == woxui.KeyDelete {
		for i, element := range structure.Elements {
			if element.Kind != common.QueryElementBlock {
				continue
			}
			start, end := queryElementRange(structure, i)
			touches := state.Selection.Start() < end && state.Selection.End() > start
			if state.Selection.Collapsed() {
				touches = event.Key == woxui.KeyBackspace && state.Selection.Focus > start && state.Selection.Focus <= end ||
					event.Key == woxui.KeyDelete && state.Selection.Focus >= start && state.Selection.Focus < end
			}
			if touches && state.Selection.Start() >= start && state.Selection.End() <= end {
				a.rememberStructuredQuery()
				next := structure.Clone()
				next.Elements = append(next.Elements[:i], next.Elements[i+1:]...)
				a.query.QueryHint = next
				a.query.QueryText = next.PlainText()
				a.editor.SetText(a.query.QueryText, false)
				a.editor.SetCaret(start)
				s.active = max(0, min(i, len(next.Elements)-1))
				if len(next.Elements) == 0 {
					a.query.QueryHint = nil
				}
				a.structuredQueryChanged()
				return true
			}
		}
		a.selectTouchedQueryBlocks()
	}
	return false
}

// replaceWholeStructuredQuery intentionally leaves pasted text unparsed.
func (a *App) replaceWholeStructuredQuery(text string) {
	a.rememberStructuredQuery()
	a.query.QueryHint = nil
	a.structured.allSelected = false
	a.structured.candidate = nil
	a.structured.suppressed = text
	a.query.QueryText = text
	a.editor.SetText(text, false)
	a.applyQueryTextChangeLocked(text)
	a.structuredQueryChanged()
}

func queryPrimaryModifier() woxui.KeyModifiers {
	if woxui.KeyModifierMeta.HasPrimary() {
		return woxui.KeyModifierMeta
	}
	return woxui.KeyModifierControl
}

// selectEntireQuery selects the same continuous document used for painting and native editing.
func (a *App) selectEntireQuery() {
	a.structured.allSelected = a.query.QueryHint != nil
	a.editor.SelectAll()
}

// queryElementRange derives rune offsets from the public structure; offsets never
// become part of the plugin protocol or include placeholder text.
func queryElementRange(structure *common.QueryHint, index int) (start, end int) {
	for i, element := range structure.Elements {
		end = start + len([]rune(element.Content()))
		if i == index {
			return start, end
		}
		start = end
	}
	return start, start
}

// selectTouchedQueryBlocks expands edits touching atomic content without affecting
// freely editable argument ranges. The same rule applies to keyboard, IME and clipboard.
func (a *App) selectTouchedQueryBlocks() {
	if a == nil {
		return
	}
	structure := a.query.QueryHint
	if structure == nil {
		return
	}
	state := a.editor.State()
	start, end := state.Selection.Start(), state.Selection.End()
	for i, element := range structure.Elements {
		if element.Kind != common.QueryElementBlock {
			continue
		}
		left, right := queryElementRange(structure, i)
		if start < right && end > left {
			start, end = min(start, left), max(end, right)
		}
	}
	if start != state.Selection.Start() || end != state.Selection.End() {
		a.editor.SetSelection(start, end)
	}
}
