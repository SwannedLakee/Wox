package plugin

import (
	"context"
	"strings"
	"wox/common"
)

// MatchQueryHint matches a complete command only; it never splits argument text.
func MatchQueryHint(text string, instances []*Instance) *common.QueryHint {
	var matched *common.QueryHint
	for _, instance := range instances {
		for _, command := range instance.GetQueryCommands() {
			if command.QueryHint == nil || command.QueryHint.Validate() != nil {
				continue
			}
			aliases := append([]string{command.Command}, command.Aliases...)
			found := false
			for _, trigger := range instance.GetTriggerKeywords() {
				for _, alias := range aliases {
					prefix := alias
					if trigger != "*" {
						prefix = trigger + " " + alias
					}
					if strings.EqualFold(strings.TrimRight(text, " "), prefix) {
						found = true
					}
				}
			}
			if !found {
				continue
			}
			if matched != nil {
				return nil
			}
			matched = command.QueryHint.Clone()
			matched.PluginId = instance.Metadata.Id
			matched.Elements = append([]common.QueryElement{{Id: "command", Kind: common.QueryElementText, Text: strings.TrimRight(text, " ") + " "}}, matched.Elements...)
			if matched.Validate() != nil {
				return nil
			}
		}
	}
	return matched
}

// ResolveQueryHint uses currently available plugins and translates the template once.
func (m *Manager) ResolveQueryHint(ctx context.Context, text string) *common.QueryHint {
	var available []*Instance
	for _, instance := range m.GetPluginInstances() {
		if instance.Setting != nil && !instance.Setting.Disabled.Get() {
			available = append(available, instance)
		}
	}
	structure := MatchQueryHint(text, available)
	if structure != nil {
		instance := m.getPluginInstance(structure.PluginId)
		for i := range structure.Elements {
			structure.Elements[i].Placeholder = common.I18nString(instance.TranslateMetadataText(ctx, structure.Elements[i].Placeholder))
		}
	}
	return structure
}
