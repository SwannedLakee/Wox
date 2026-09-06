package common

import (
	"encoding/json"
	"testing"
)

func TestQueryHintRoundTripAndIsolation(t *testing.T) {
	s := &QueryHint{Elements: []QueryElement{{Id: "command", Kind: QueryElementText, Text: "gh issues "}, {Id: "repo", Kind: QueryElementArgument, Placeholder: "Repository"}, {Id: "issue", Kind: QueryElementBlock, Value: "owner/repo#6"}}}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	if s.PlainText() != "gh issues owner/repo#6" {
		t.Fatal(s.PlainText())
	}
	clone := s.Clone()
	clone.Elements[1].Value = "changed"
	if s.Argument("repo") != "" {
		t.Fatal("clone mutated the template")
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var restored QueryHint
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.PlainText() != s.PlainText() {
		t.Fatal("round trip lost content")
	}
	clone.Elements[1].Id = "command"
	if clone.Validate() == nil {
		t.Fatal("duplicate IDs accepted")
	}
	clone.Elements[1].Id = "repo"
	clone.Elements[1].Kind = "unknown"
	if clone.Validate() == nil {
		t.Fatal("unknown kind accepted")
	}
}
