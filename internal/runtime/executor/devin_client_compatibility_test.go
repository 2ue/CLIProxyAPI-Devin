package executor

import (
	"context"
	"github.com/MoeForge/moego/ojson"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/runtime/executor/helps"
	"net/http"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v8/sdk/translator"
)

func TestDevinNativeClientThinkingControls(t *testing.T) {
	exec := NewDevinExecutor(&config.Config{})
	auth := &cliproxyauth.Auth{ID: t.Name(), Provider: "devin", Attributes: map[string]string{"api_key": "test-key"}}
	for _, tc := range []struct {
		name    string
		format  sdktranslator.Format
		model   string
		payload string
		want    string
	}{
		{"Claude disabled beats effort", sdktranslator.FormatClaude, "devin/gpt-5-6-sol", `{"max_tokens":4096,"thinking":{"type":"disabled"},"output_config":{"effort":"max"},"messages":[{"role":"user","content":"hello"}]}`, "gpt-5-6-sol-none"},
		{"Claude between tools beats effort", sdktranslator.FormatClaude, "devin/gpt-5-6-sol", `{"max_tokens":4096,"thinking":{"type":"between_tools"},"output_config":{"effort":"max"},"messages":[{"role":"user","content":"hello"}]}`, "gpt-5-6-sol-none"},
		{"model suffix beats disabled body", sdktranslator.FormatClaude, "devin/gpt-5-6-sol(high)", `{"thinking":{"type":"disabled"},"messages":[{"role":"user","content":"hello"}]}`, "gpt-5-6-sol-high"},
		{"Claude numeric budget suffix", sdktranslator.FormatClaude, "devin/claude-opus-5-5(2048)", `{"thinking":{"type":"enabled","budget_tokens":64000},"messages":[{"role":"user","content":"hello"}]}`, "claude-opus-5-5-low"},
		{"Gemini thinking level", sdktranslator.FormatGemini, "devin/gemini-3-8-flash", `{"contents":[{"role":"user","parts":[{"text":"hello"}]}],"generationConfig":{"thinkingConfig":{"thinkingLevel":"low"}}}`, "gemini-3-8-flash-low"},
		{"Gemini zero budget", sdktranslator.FormatGemini, "devin/gemini-3-8-flash", `{"contents":[{"role":"user","parts":[{"text":"hello"}]}],"generationConfig":{"thinkingConfig":{"thinkingBudget":0}}}`, "gemini-3-8-flash-low"},
		{"Gemini positive budget", sdktranslator.FormatGemini, "devin/gemini-3-8-flash", `{"contents":[{"role":"user","parts":[{"text":"hello"}]}],"generationConfig":{"thinkingConfig":{"thinkingBudget":2048}}}`, "gemini-3-8-flash-low"},
		{"Gemini minimum supported level", sdktranslator.FormatGemini, "devin/gemini-3-5-flash", `{"contents":[{"role":"user","parts":[{"text":"hello"}]}],"generationConfig":{"thinkingConfig":{"thinkingBudget":0}}}`, "gemini-3-5-flash-minimal"},
		{"Codex reasoning effort", sdktranslator.FormatOpenAIResponse, "devin/gpt-6-1-sol", `{"input":"hello","reasoning":{"effort":"xhigh"}}`, "gpt-6-1-sol-xhigh"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := cliproxyexecutor.Request{Model: tc.model, Payload: []byte(tc.payload)}
			_, uid, _, err := exec.prepareDevinHTTPRequest(context.Background(), auth, req, cliproxyexecutor.Options{SourceFormat: tc.format})
			if err != nil || uid != tc.want {
				t.Fatalf("upstream UID = %q, err = %v, want %q", uid, err, tc.want)
			}
		})
	}
}

func TestDevinStreamRefusalPreservesRequestScope(t *testing.T) {
	err := devinStreamTrailerError(t, nil, "permission_denied", "Your request was blocked by our content policy.")
	status, ok := err.(statusErr)
	if !ok || !status.IsRequestScoped() || status.StatusCode() != http.StatusForbidden {
		t.Fatalf("stream refusal = %T %v, want request-scoped 403", err, err)
	}
}

func TestDevinContentRefusalIsRequestScoped(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    int
		message string
		want    bool
	}{
		{"Devin content refusal", 403, "devin upstream error (permission_denied): Your request was blocked by our content policy.", true},
		{"Anthropic content refusal", 400, "This request was refused by Anthropic's content policy.", true},
		{"account entitlement", 403, "model access is not allowed for this account", false},
		{"credential failure", 401, "invalid token", false},
		{"rate limit", 429, "rate limit exceeded", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, err := range []statusErr{devinTrailerStatusError(tc.code, tc.message, nil), newDevinStatusErrorForAuth(tc.code, http.Header{}, []byte(tc.message), nil)} {
				if err.IsRequestScoped() != tc.want || err.StatusCode() != tc.code || err.IsCredentialScoped() {
					t.Fatalf("classification = %+v, want request scoped %v with status %d", err, tc.want, tc.code)
				}
			}
		})
	}
}

func TestDevinHiddenNonstreamThoughtKeepsClaudeSignature(t *testing.T) {
	raw := helps.DevinHideThinkingText([]byte(`{"id":"test","model":"devin/claude-opus-5-5","steps":[{"type":"thought","content":[{"type":"text","text":"private thought"}],"signature":"valid-signature"},{"type":"model_output","content":[{"type":"text","text":"answer"}]}]}`))
	var state any
	out := sdktranslator.TranslateNonStream(context.Background(), sdktranslator.FormatInteractions, sdktranslator.FormatClaude, "devin/claude-opus-5-5", nil, nil, raw, &state)
	if ojson.GetBytes(out, "content.0.thinking").String() != "" || ojson.GetBytes(out, "content.0.signature").String() != "valid-signature" || ojson.GetBytes(out, "content.1.text").String() != "answer" {
		t.Fatalf("hidden thought translation lost signature or answer: %s", out)
	}
}
