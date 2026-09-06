package preview

import (
	"testing"

	woxcomponent "wox/ui/launcher/component"
	woxui "wox/ui/runtime"
	woxwidget "wox/ui/widget"
)

func TestRequirementPreviewMessageRendersMarkdownLinks(t *testing.T) {
	opened := ""
	view := RequirementPreviewView(RequirementPreviewProps{
		Width: 420, Height: 280, Title: "Spotify needs configuration",
		Message: "Create an app at [Spotify Dashboard](https://developer.spotify.com/dashboard).",
		Theme:   woxcomponent.Theme{}, OnOpenLink: func(target string) { opened = target },
	})
	root := view.(woxwidget.Container)
	column := root.Child.(woxwidget.Flex)
	message := column.Children[1].(woxwidget.Container).Child.(woxwidget.Flex)
	paragraph := message.Children[0].(woxwidget.Wrap)
	for _, child := range paragraph.Children {
		if link, ok := child.(woxwidget.Semantics); ok && link.Role == woxui.AccessibilityRoleLink {
			_ = link.OnAction(woxui.AccessibilityActionActivate, "")
			break
		}
	}
	if opened != "https://developer.spotify.com/dashboard" {
		t.Fatalf("opened link = %q, want the requirement message target", opened)
	}
}
