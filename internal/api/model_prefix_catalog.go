package api

import (
	"sort"
	"strings"
	"unicode"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
)

type modelPrefixEntry struct{ public, route string }

func modelPrefixProtocolModels(protocol string) map[string]map[string]any {
	result := map[string]map[string]any{}
	for _, m := range registry.GetGlobalRegistry().GetAvailableModels(protocol) {
		id, _ := m["id"].(string)
		if protocol == "gemini" {
			id, _ = m["name"].(string)
			id = strings.TrimPrefix(id, "models/")
		}
		if id != "" {
			result[id] = m
		}
	}
	return result
}

func clonePrefixModel(original map[string]any) map[string]any {
	result := make(map[string]any, len(original))
	for k, v := range original {
		result[k] = v
	}
	return result
}

func buildModelPrefixCatalog(prefix string, cfg *config.Config) []modelPrefixEntry {
	canonical := standardPrefixModelIndex()
	models := modelPrefixProtocolModels("openai")
	selected := map[string]string{}
	ids := make([]string, 0, len(models))
	for id := range models {
		if strings.HasPrefix(id, prefix+"/") {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		name := strings.TrimPrefix(id, prefix+"/")
		public := name
		info := registry.LookupModelInfo(id)
		// Existing configured aliases are deliberate, even when they are not vendor names.
		if info == nil || (info.OAuthAlias == "" && !info.UserDefined) {
			candidate := name
			if info != nil && info.MetadataModelID != "" {
				candidate = strings.TrimPrefix(info.MetadataModelID, prefix+"/")
			}
			if official := canonical[normalizedPrefixModel(candidate)]; official != "" {
				public = official
			}
			if prefix == "devin" && name == "claude-5-fable" {
				public = "claude-fable-5"
			}
		}
		if previous := selected[public]; previous == "" || name == public {
			selected[public] = id
		}
	}
	if cfg != nil {
		for public, target := range cfg.ModelPrefixAliases[prefix] {
			public = strings.TrimSpace(public)
			target = strings.TrimSpace(target)
			if !strings.HasPrefix(target, prefix+"/") {
				target = prefix + "/" + target
			}
			if public != "" && models[target] != nil {
				for previous, route := range selected {
					if route == target && cfg.ModelPrefixAliases[prefix][previous] == "" {
						delete(selected, previous)
					}
				}
				selected[public] = target
			}
		}
	}
	result := make([]modelPrefixEntry, 0, len(selected))
	for public, route := range selected {
		result = append(result, modelPrefixEntry{public, route})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].public < result[j].public })
	return result
}

// Catalog identities are authoritative; punctuation differences do not change the model.
// Ambiguous normalized names are left unchanged rather than guessed.
func standardPrefixModelIndex() map[string]string {
	result := map[string]string{}
	// Official OpenAI IDs retained after older entries leave the active plan catalog.
	// https://developers.openai.com/api/docs/models
	for _, id := range []string{"gpt-5.4", "gpt-5.4-mini", "gpt-5.5"} {
		result[normalizedPrefixModel(id)] = id
	}
	ambiguous := map[string]bool{}
	groups := [][]*registry.ModelInfo{registry.GetClaudeModels(), registry.GetGeminiModels(), registry.GetCodexFreeModels(), registry.GetCodexTeamModels(), registry.GetCodexPlusModels(), registry.GetCodexProModels(), registry.GetKimiModels(), registry.GetXAIModels(), registry.GetMetaModels()}
	for _, models := range groups {
		for _, m := range models {
			if m == nil {
				continue
			}
			key := normalizedPrefixModel(m.ID)
			if previous := result[key]; previous != "" && previous != m.ID {
				ambiguous[key] = true
			}
			result[key] = m.ID
		}
	}
	for key := range ambiguous {
		delete(result, key)
	}
	return result
}

func normalizedPrefixModel(id string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, id)
}
