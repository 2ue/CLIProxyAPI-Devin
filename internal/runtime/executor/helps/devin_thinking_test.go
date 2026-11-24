package helps

import (
	"github.com/MoeForge/moego/ojson"
	"testing"
)

func TestDevinHideThinkingTextPreservesSignaturesAndTools(t *testing.T) {
	raw := []byte(`{"steps":[{"type":"thought","content":[{"type":"text","text":"internal reasoning"}],"signature":"sig"},{"type":"function_call","arguments":{"text":"tool input"}},{"type":"model_output","content":[{"text":"answer"}]}]}`)
	got := DevinHideThinkingText(raw)
	if ojson.GetBytes(got, "steps.0.content").String() != "" || ojson.GetBytes(got, "steps.0.signature").String() != "sig" || ojson.GetBytes(got, "steps.1.arguments.text").String() != "tool input" || ojson.GetBytes(got, "steps.2.content.0.text").String() != "answer" {
		t.Fatalf("invalid hidden response: %s", got)
	}
	event := DevinHideThinkingText([]byte(`{"event_type":"step.delta","delta":{"type":"thought_summary","text":"internal","content":{"text":"internal"}}}`))
	if ojson.GetBytes(event, "delta.text").String() != "" || ojson.GetBytes(event, "delta.content.text").String() != "" {
		t.Fatalf("thought leaked: %s", event)
	}
}
