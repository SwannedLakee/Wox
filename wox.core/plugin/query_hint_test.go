package plugin

import (
	"testing"
	"wox/common"
)

func TestMatchQueryHint(t *testing.T) {
	template := &common.QueryHint{Elements: []common.QueryElement{{Id: "volume", Kind: "argument", Placeholder: "Volume"}}}
	instance := &Instance{Metadata: Metadata{Id: "sys", TriggerKeywords: []string{"*"}, Commands: []MetadataCommand{{Command: "set-volume", Aliases: []string{"set volume", "volume"}, QueryHint: template}}}}
	for _, text := range []string{"set volume", "set volume ", "volume ", "SET VOLUME"} {
		got := MatchQueryHint(text, []*Instance{instance})
		if got == nil || got.PluginId != "sys" || len(got.Elements) != 2 {
			t.Fatalf("%q did not create slots", text)
		}
		got.Elements[1].Value = "50"
		if template.Argument("volume") != "" {
			t.Fatal("instance mutated declaration")
		}
	}
	for _, text := range []string{"set volum", "set volumex", "set volume 50"} {
		if MatchQueryHint(text, []*Instance{instance}) != nil {
			t.Fatalf("%q was parsed unexpectedly", text)
		}
	}
	if MatchQueryHint("volume", []*Instance{instance, instance}) != nil {
		t.Fatal("ambiguous match selected an owner")
	}
	instance.Metadata.TriggerKeywords = []string{"gh"}
	instance.Metadata.Commands[0].Command = "issues"
	if MatchQueryHint("gh issues ", []*Instance{instance}) == nil {
		t.Fatal("plugin trigger was not matched")
	}
}
