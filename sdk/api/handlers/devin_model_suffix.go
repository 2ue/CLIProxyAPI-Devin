package handlers

import (
	"slices"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/util"
)

func normalizeDevinRequestSuffix(model string) string {
	if !strings.HasPrefix(model, "devin/") || len(util.GetProviderName(model)) > 0 {
		return model
	}
	for _, suffix := range []string{"-thinking", "-think", "-minimal", "-medium", "-xhigh", "-ultra", "-high", "-none", "-auto", "-max", "-low"} {
		if !strings.HasSuffix(model, suffix) {
			continue
		}
		base := strings.TrimSuffix(model, suffix)
		if !slices.Contains(util.GetProviderName(base), "devin") {
			continue
		}
		level := strings.TrimPrefix(suffix, "-")
		if level == "think" || level == "thinking" {
			level = "high"
		}
		return base + "(" + level + ")"
	}
	return model
}
