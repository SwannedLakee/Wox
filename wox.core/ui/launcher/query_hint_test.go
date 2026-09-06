package launcher

import (
	"context"
	"testing"
	"wox/common"
	woxui "wox/ui/runtime"
)

func TestSelectEntireQueryHintThenType(t *testing.T) {
	a := &App{editor: woxui.NewTextEditor(""), lifecycleCtx: context.Background(), query: newInputQuery("set volume ")}
	a.installQueryHint(&common.QueryHint{Elements: []common.QueryElement{
		{Id: "command", Kind: "text", Text: "set volume "},
		{Id: "volume", Kind: "argument", Value: "30"},
	}})
	// Reopening with SelectAll must select the document, not only the focused slot.
	a.selectEntireQuery()
	if !a.queryHintEditorState.allSelected || a.editor.SelectedText() != "set volume 30" {
		t.Fatal("launcher selection did not include the query document with hints")
	}
	a.onTextInput(woxui.TextInputEvent{Kind: woxui.TextInputCommit, Text: "new search"})
	if a.query.QueryHint != nil || a.query.QueryText != "new search" || a.editor.State().Text != "new search" {
		t.Fatal("typing after reopen must replace the command and all arguments")
	}
	a.selectEntireQuery()
	if a.queryHintEditorState.allSelected || a.editor.SelectedText() != "new search" {
		t.Fatal("plain queries must retain ordinary select-all behavior")
	}
}

func TestQueryHintEditing(t *testing.T) {
	a := &App{editor: woxui.NewTextEditor(""), lifecycleCtx: context.Background(), query: newInputQuery("gh issues ")}
	s := &common.QueryHint{Elements: []common.QueryElement{
		{Id: "command", Kind: "text", Text: "gh issues "},
		{Id: "repo", Kind: "argument", Value: "owner/repo"},
		{Id: "space", Kind: "text", Text: " "},
		{Id: "issue", Kind: "argument", Value: "6"},
		{Id: "entity", Kind: "block", Value: "selected"},
	}}
	a.installQueryHint(s)
	key := func(k woxui.Key, mods woxui.KeyModifiers) {
		t.Helper()
		if !a.onQueryHintKey(woxui.KeyEvent{Key: k, Modifiers: mods, Down: true}) {
			t.Fatalf("unhandled %s", k)
		}
	}
	if a.editor.State().Text != s.PlainText() {
		t.Fatal("first argument not focused")
	}
	key(woxui.KeyTab, 0)
	if a.queryHintEditorState.active != 3 || a.editor.SelectedText() != "6" {
		t.Fatal("Tab did not skip separator")
	}
	a.editor.InsertText("42")
	text := a.updateQueryHintText(a.editor.State().Text)
	a.query.QueryText = text
	if a.query.QueryHint.Argument("issue") != "42" || s.Argument("issue") != "6" {
		t.Fatal("slot edit leaked or was lost")
	}
	key(woxui.KeyTab, 0)
	key(woxui.KeyBackspace, 0)
	if len(a.query.QueryHint.Elements) != 4 {
		t.Fatal("block was not deleted")
	}
	mods := woxui.KeyModifierControl
	if woxui.KeyModifierMeta.HasPrimary() {
		mods = woxui.KeyModifierMeta
	}
	key(woxui.Key("z"), mods)
	if len(a.query.QueryHint.Elements) != 5 {
		t.Fatal("undo lost the block")
	}
	key(woxui.Key("z"), mods)
	if a.query.QueryHint.Argument("issue") != "6" {
		t.Fatal("undo lost argument value")
	}
}

func TestQueryHintContinuousEditing(t *testing.T) {
	a := &App{editor: woxui.NewTextEditor(""), lifecycleCtx: context.Background(), query: newInputQuery("set volume ")}
	a.installQueryHint(&common.QueryHint{Elements: []common.QueryElement{{Id: "command", Kind: "text", Text: "set volume "}, {Id: "volume", Kind: "argument", Value: "50", Placeholder: "Volume"}}})
	edit := func(text string) {
		t.Helper()
		a.editor.InsertText(text)
		a.query.QueryText = a.updateQueryHintText(a.editor.State().Text)
	}
	a.editor.SetSelection(11, 13)
	edit("30")
	if a.query.QueryHint.Argument("volume") != "30" || a.editor.State().Text != "set volume 30" {
		t.Fatal("continuous argument replacement failed")
	}
	// Selecting across the command and argument must preserve exactly the user's edit.
	a.editor.SetSelection(4, 12)
	if a.editor.SelectedText() != "volume 3" {
		t.Fatal("cross-element selection was split")
	}
	edit("other ")
	if a.query.QueryHint != nil || a.query.QueryText != "set other 0" {
		t.Fatal("cross-element replacement lost text or retained stale hint")
	}
	a.onQueryHintKey(woxui.KeyEvent{Key: woxui.Key("z"), Modifiers: queryPrimaryModifier(), Down: true})
	if a.query.QueryHint == nil || a.editor.State().Text != "set volume 30" {
		t.Fatal("undo did not restore hint and full text")
	}
}

func TestQueryHintUnicodeAndOrdinaryBackspace(t *testing.T) {
	a := &App{editor: woxui.NewTextEditor(""), lifecycleCtx: context.Background(), query: newInputQuery("timer ")}
	a.installQueryHint(&common.QueryHint{Elements: []common.QueryElement{
		{Id: "command", Kind: "text", Text: "timer "}, {Id: "duration", Kind: "argument", Value: "5m"},
		{Id: "separator", Kind: "text", Text: " "}, {Id: "note", Kind: "argument", Value: "泡茶"},
	}})
	a.focusQueryElement(3)
	a.editor.InsertText("喝水")
	a.query.QueryText = a.updateQueryHintText(a.editor.State().Text)
	if a.query.QueryHint.Argument("note") != "喝水" {
		t.Fatal("rune offsets corrupted argument")
	}
	event := woxui.KeyEvent{Key: woxui.KeyBackspace, Down: true}
	if a.onQueryHintKey(event) {
		t.Fatal("argument intercepted ordinary Backspace")
	}
	a.editor.HandleKey(event)
	a.query.QueryText = a.updateQueryHintText(a.editor.State().Text)
	if a.query.QueryHint.Argument("note") != "喝" {
		t.Fatal("Backspace did not delete one character")
	}
}

func TestQueryHintWholeReplacementUndo(t *testing.T) {
	a := &App{editor: woxui.NewTextEditor(""), lifecycleCtx: context.Background(), query: newInputQuery("volume ")}
	a.installQueryHint(&common.QueryHint{Elements: []common.QueryElement{{Id: "command", Kind: "text", Text: "volume "}, {Id: "volume", Kind: "argument", Value: "50"}}})
	a.replaceWholeQueryHint("set volume 75")
	if a.query.QueryHint != nil || a.query.QueryText != "set volume 75" {
		t.Fatal("whole paste was parsed or lost")
	}
	a.onQueryHintKey(woxui.KeyEvent{Key: woxui.Key("z"), Modifiers: queryPrimaryModifier(), Down: true})
	if a.query.QueryHint == nil || a.query.QueryHint.Argument("volume") != "50" {
		t.Fatal("one undo must restore the complete document")
	}
}

func TestInstallBlockHintLeavesCaretAfterBlock(t *testing.T) {
	a := &App{editor: woxui.NewTextEditor("")}
	hint := &common.QueryHint{Elements: []common.QueryElement{
		{Id: "command", Kind: "text", Text: "gh issues "},
		{Id: "issue", Kind: "block", Value: "owner/仓库#6"},
	}}
	a.installQueryHint(hint)
	if a.editor.SelectedText() != "" || a.editor.State().Selection.Focus != len([]rune(hint.PlainText())) {
		t.Fatal("block hint must leave an unselected caret after its value")
	}
	a.editor.InsertText(" more")
	if a.editor.State().Text != hint.PlainText()+" more" {
		t.Fatal("typing must append after the block instead of replacing it")
	}
}
