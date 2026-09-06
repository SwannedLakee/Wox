package launcher

import (
	"strings"
	"wox/common"
	launcherview "wox/ui/launcher/view"
	woxwidget "wox/ui/widget"
)

// queryHintView decorates the ordinary editor without introducing separate
// hit targets, padding, or text layout. Selection and IME retain one coordinate space.
func (a *App) queryHintView(snapshot viewSnapshot, width, height, lineHeight float32) woxwidget.Widget {
	snapshot.completionHint = nil
	props := a.queryViewProps(snapshot, width, height, lineHeight)
	measure := func(text string) float32 {
		metrics, _ := a.window.MeasureText(text, props.Style)
		return metrics.Size.Width
	}
	// Composition temporarily changes display offsets; resume marks after commit.
	if props.State.Composition == "" {
		offset := 0
		text := []rune(props.State.Text)
		for _, element := range snapshot.hint.Elements {
			start, end := offset, offset+len([]rune(element.Content()))
			offset = end
			if element.Kind == common.QueryElementText {
				continue
			}
			start, end = min(start, len(text)), min(end, len(text))
			if start == end {
				if props.CompletionSuffix == "" && end == len(text) {
					hints := []string{}
					for _, e := range snapshot.hint.Elements {
						if e.Kind == common.QueryElementArgument && e.Value == "" {
							hints = append(hints, string(e.Placeholder))
						}
					}
					props.CompletionSuffix = strings.Join(hints, "   ")
					if snapshot.queryHintCandidate {
						props.CompletionSuffix = " " + props.CompletionSuffix
					}
					props.TextWidth += measure(props.CompletionSuffix)
				}
				continue
			}
			lineStart := 0
			for lineIndex, line := range props.Lines {
				lineEnd := lineStart + len([]rune(line.Text))
				left, right := max(start, lineStart), min(end, lineEnd)
				if left < right {
					runes := []rune(line.Text)
					x := measure(string(runes[:left-lineStart]))
					rightX := measure(string(runes[:right-lineStart]))
					props.Marks = append(props.Marks, launcherview.LauncherQueryMark{Line: lineIndex, X: x, Width: rightX - x,
						Active: props.State.Selection.Collapsed() && props.State.Selection.Focus >= start && props.State.Selection.Focus <= end})
				}
				lineStart = lineEnd + 1
			}
		}
	}
	return launcherview.LauncherQueryBoundary(props)
}
