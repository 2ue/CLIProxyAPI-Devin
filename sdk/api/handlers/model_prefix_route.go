package handlers

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/thinking"
	"golang.org/x/net/context"
)

// ModelPrefixContextKey and ModelPrefixRoutesContextKey carry an authenticated scoped catalog.
const ModelPrefixContextKey = "cliproxy.model_prefix"
const ModelPrefixRoutesContextKey = "cliproxy.model_prefix_routes"

func prefixedModelRoute(ctx context.Context, model string) (modelRouteDecision, bool) {
	if ctx == nil {
		return modelRouteDecision{}, false
	}
	c, ok := ctx.Value("gin").(*gin.Context)
	if !ok || c == nil {
		return modelRouteDecision{}, false
	}
	prefix := c.GetString(ModelPrefixContextKey)
	if prefix == "" {
		return modelRouteDecision{}, false
	}
	value, _ := c.Get(ModelPrefixRoutesContextKey)
	routes, _ := value.(map[string]string)
	model = strings.TrimSpace(model)
	target := routes[model]
	base := target
	if target == "" {
		parsed := thinking.ParseSuffix(model)
		if parsed.HasSuffix && routes[parsed.ModelName] != "" {
			target = routes[parsed.ModelName] + "(" + parsed.RawSuffix + ")"
			base = routes[parsed.ModelName]
		}
	}
	if target == "" {
		// Preserve existing provider-specific suffixes without guessing model identities.
		keys := make([]string, 0, len(routes))
		for public := range routes {
			keys = append(keys, public)
		}
		sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
		for _, public := range keys {
			if !strings.HasPrefix(model, public) {
				continue
			}
			suffix := strings.TrimPrefix(model, public)
			switch suffix {
			case "-think", "-thinking", "-minimal", "-low", "-medium", "-high", "-xhigh", "-max", "-ultra", "-none", "-auto":
				level := strings.TrimPrefix(suffix, "-")
				if level == "think" || level == "thinking" {
					level = "high"
				}
				target = routes[public] + "(" + level + ")"
				base = routes[public]
			}
			if target != "" {
				break
			}
		}
	}
	if target == "" {
		target = prefix + "/" + model
		base = target
	}
	return modelRouteDecision{Model: target, ScopedPrefix: true, ScopedBase: base}, true
}
