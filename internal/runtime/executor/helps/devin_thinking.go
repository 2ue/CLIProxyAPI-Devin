package helps

import (
	"strconv"
	"strings"

	"github.com/MoeForge/moego/ojson"
)

// DevinGeminiThinkingControl preserves explicit zero budgets that the shared
// Interactions translation cannot distinguish from an absent budget.
func DevinGeminiThinkingControl(payload []byte) (level string, budget int, ok bool) {
	root := ojson.GetBytesNoCopy(payload, "generationConfig.thinkingConfig")
	if !root.Exists() {
		return "", 0, false
	}
	if value := root.Get("thinkingLevel"); value.Type == ojson.String && strings.TrimSpace(value.String()) != "" {
		return strings.ToLower(strings.TrimSpace(value.String())), 0, true
	}
	value := root.Get("thinkingBudget")
	if !value.Exists() || value.Type != ojson.Number {
		return "", 0, false
	}
	switch budget = int(value.Int()); {
	case budget < 0:
		return "auto", 0, true
	case budget == 0:
		return "none", 0, true
	default:
		return "", budget, true
	}
}

// DevinHideThinkingText removes display text without dropping signatures or tool content.
func DevinHideThinkingText(payload []byte) []byte {
	doc := ojson.NewDoc(payload)
	if doc.Get("delta.type").String() == "thought_summary" {
		_ = doc.Set("delta.text", "")
		if doc.Get("delta.content").Exists() {
			_ = doc.Set("delta.content.text", "")
		}
	}
	for _, path := range []string{"steps", "interaction.steps"} {
		for index, step := range doc.Get(path).Array() {
			if step.Get("type").String() == "thought" {
				_ = doc.Set(path+"."+strconv.Itoa(index)+".content", "")
			}
		}
	}
	return doc.Bytes()
}
