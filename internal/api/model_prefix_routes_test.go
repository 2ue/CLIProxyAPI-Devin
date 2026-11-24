package api

import (
	"context"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	coreexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MoeForge/moego/ojson"
	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
)

func TestModelPrefixCatalogRootCompatibilityAndAuthentication(t *testing.T) {
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(t.Name(), "devin", []*registry.ModelInfo{{ID: "devin/claude-test-scope", Name: "devin/claude-test-scope", Type: "devin", ContextLength: 200000, MaxCompletionTokens: 8192}})
	reg.RegisterClient(t.Name()+"-other", "codex", []*registry.ModelInfo{{ID: "other-only-scope", Type: "codex"}})
	t.Cleanup(func() { reg.UnregisterClient(t.Name()); reg.UnregisterClient(t.Name() + "-other") })
	s := newTestServer(t)
	tests := []struct{ path, header, want, exclude string }{
		{"/v1/models", "", "devin/claude-test-scope", ""},
		{"/llm/devin/v1/models", "", "claude-test-scope", "devin/claude-test-scope"},
		{"/llm/devin/v1/models", "anthropic", "claude-test-scope", "claude-fable-5-dd"},
		{"/llm/devin/v1/models/claude-test-scope", "", "claude-test-scope", "devin/"},
		{"/llm/devin/v1/models?client_version=0.171.0", "", `"slug":"claude-test-scope"`, `"slug":"devin/claude-test-scope"`},
		{"/llm/devin/v1beta/models", "", "models/claude-test-scope", "models/devin/"},
		{"/llm/devin/v1beta/models/claude-test-scope", "", "models/claude-test-scope", "models/devin/"},
	}
	for _, tc := range tests {
		for _, key := range []string{"", "test-key"} {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if key != "" {
				req.Header.Set("Authorization", "Bearer "+key)
			}
			if tc.header == "anthropic" {
				req.Header.Set("anthropic-version", "2023-06-01")
			}
			rr := httptest.NewRecorder()
			s.engine.ServeHTTP(rr, req)
			if key == "" {
				if rr.Code != 401 {
					t.Fatalf("unauthenticated %s = %d", tc.path, rr.Code)
				}
				continue
			}
			body := rr.Body.String()
			if rr.Code != 200 || !strings.Contains(body, tc.want) || (tc.exclude != "" && strings.Contains(body, tc.exclude)) {
				t.Fatalf("%s = %d %s", tc.path, rr.Code, body)
			}
			if strings.HasPrefix(tc.path, "/llm/") && strings.Contains(body, "other-only-scope") {
				t.Fatalf("other provider leaked: %s", body)
			}
		}
	}
}

func TestModelPrefixResponseOnlyChangesModelMetadata(t *testing.T) {
	raw := []byte(`{"model":"devin/claude-test","message":{"model":"devin/claude-test","content":[{"text":"devin/claude-test"}]},"response":{"model":"devin/gpt-test"},"modelVersion":"devin/gemini-test","arguments":{"model":"devin/do-not-change"}}`)
	got := prefixPublicResponseModel(raw, "devin", nil)
	for path, want := range map[string]string{"model": "claude-test", "message.model": "claude-test", "response.model": "gpt-test", "modelVersion": "gemini-test", "message.content.0.text": "devin/claude-test", "arguments.model": "devin/do-not-change"} {
		if actual := ojson.GetBytes(got, path).String(); actual != want {
			t.Fatalf("%s=%q want %q", path, actual, want)
		}
	}
}

func TestModelPrefixStreamingFragmentationAndFlush(t *testing.T) {
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	w := &modelPrefixResponseWriter{ResponseWriter: c.Writer, prefix: "devin"}
	w.Header().Set("Content-Type", "text/event-stream")
	parts := []string{"event: message_start\r\ndata: {\"message\":{\"mo", "del\":\"devin/claude-test\"}}\r\n\r\n", "data: {\"delta\":{\"text\":\"devin/keep\"}}\n\ndata: [DONE]\n\n"}
	for _, part := range parts {
		if _, err := w.WriteString(part); err != nil {
			t.Fatal(err)
		}
		w.Flush()
	}
	w.finish()
	got := rr.Body.String()
	if !rr.Flushed || strings.Contains(got, "devin/claude-test") || !strings.Contains(got, "devin/keep") || !strings.Contains(got, "data: [DONE]\n\n") {
		t.Fatalf("broken SSE: %q", got)
	}
}

func TestModelPrefixBodyLimit(t *testing.T) {
	cfg := parseTestServerConfigYAML(t, "config-version: 8\nserver:\n  max-request-body-bytes: 64\naccess:\n  api-keys:\n    - test-key\n")
	s := newTestServerWithConfig(t, cfg)
	req := httptest.NewRequest(http.MethodPost, "/llm/devin/v1/messages", strings.NewReader(strings.Repeat("x", 128)))
	req.Header.Set("Authorization", "Bearer test-key")
	rr := httptest.NewRecorder()
	s.engine.ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d", rr.Code)
	}
	if !isExampleAPIKeySafeModeProxyPath("/llm/devin/v1/models") {
		t.Fatal("safe-mode guard missed scoped route")
	}
}

func TestModelPrefixCustomProviderAndStandardNameMapping(t *testing.T) {
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(t.Name(), "custom-openai", []*registry.ModelInfo{
		{ID: "team/gpt-6-1-sol"},
		{ID: "team/gemini-3-8-flash"},
		{ID: "team/vendor-deployment-123", UserDefined: true},
		{ID: "team/my-configured-alias", OAuthAlias: "my-configured-alias", MetadataModelID: "gpt-6.1-sol"},
	})
	t.Cleanup(func() { reg.UnregisterClient(t.Name()) })
	cfg := parseTestServerConfigYAML(t, "config-version: 8\nrouting:\n  model-prefix-aliases:\n    team:\n      claude-opus-5-5: vendor-deployment-123\naccess:\n  api-keys:\n    - test-key\n")
	s := newTestServerWithConfig(t, cfg)
	for _, path := range []string{"/llm/team/v1/models", "/llm/team/v1beta/models"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer test-key")
		rr := httptest.NewRecorder()
		s.engine.ServeHTTP(rr, req)
		body := rr.Body.String()
		for _, name := range []string{"gpt-6.1-sol", "gemini-3.8-flash", "claude-opus-5-5", "my-configured-alias"} {
			if !strings.Contains(body, name) {
				t.Fatalf("missing %s: %s", name, body)
			}
		}
		if strings.Contains(body, `"id":"vendor-deployment-123"`) {
			t.Fatalf("configured canonical name did not replace deployment name: %s", body)
		}
		if strings.Contains(body, "team/") {
			t.Fatalf("prefix leaked in scoped catalog: %s", body)
		}
	}
	root := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	root.Header.Set("Authorization", "Bearer test-key")
	rr := httptest.NewRecorder()
	s.engine.ServeHTTP(rr, root)
	if !strings.Contains(rr.Body.String(), "team/gpt-6-1-sol") || strings.Contains(rr.Body.String(), `"id":"gpt-6.1-sol"`) {
		t.Fatalf("root renamed: %s", rr.Body.String())
	}
}

type prefixCaptureExecutor struct {
	codexSearchCaptureExecutor
	model string
}

func (e *prefixCaptureExecutor) Identifier() string { return "prefix-test-provider" }
func (e *prefixCaptureExecutor) Execute(_ context.Context, _ *auth.Auth, req coreexecutor.Request, _ coreexecutor.Options) (coreexecutor.Response, error) {
	e.model = req.Model
	return coreexecutor.Response{Payload: []byte(`{"id":"test","model":"team/gpt-6-1-sol","choices":[{"message":{"role":"assistant","content":"team/do not replace"},"finish_reason":"stop"}]}`)}, nil
}

func TestModelPrefixExecutesCustomProviderAndRejectsEscape(t *testing.T) {
	server := newTestServer(t)
	executor := &prefixCaptureExecutor{}
	server.handlers.AuthManager.RegisterExecutor(executor)
	credential := &auth.Auth{ID: t.Name(), Provider: executor.Identifier(), Prefix: "team", Status: auth.StatusActive, Attributes: map[string]string{"api_key": "test"}}
	if _, err := server.handlers.AuthManager.Register(context.Background(), credential); err != nil {
		t.Fatal(err)
	}
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(credential.ID, credential.Provider, []*registry.ModelInfo{{ID: "team/gpt-6-1-sol"}})
	t.Cleanup(func() { reg.UnregisterClient(credential.ID) })
	for _, tc := range []struct {
		model  string
		status int
	}{{"gpt-6.1-sol", 200}, {"other/gpt-6.1-sol", 404}} {
		req := httptest.NewRequest(http.MethodPost, "/llm/team/v1/chat/completions", strings.NewReader(`{"model":"`+tc.model+`","messages":[{"role":"user","content":"hello"}]}`))
		req.Header.Set("Authorization", "Bearer test-key")
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		server.engine.ServeHTTP(rr, req)
		if rr.Code != tc.status {
			t.Fatalf("%s status=%d %s", tc.model, rr.Code, rr.Body.String())
		}
		if tc.status == 200 && (executor.model != "gpt-6-1-sol" || ojson.GetBytes(rr.Body.Bytes(), "model").String() != "gpt-6.1-sol" || !strings.Contains(rr.Body.String(), "team/do not replace")) {
			t.Fatalf("route/response mismatch model=%q body=%s", executor.model, rr.Body.String())
		}
	}
}
