package handlers

import (
	"context"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
)

func TestModelPrefixRoutePreservesRootAndExistingProviderPool(t *testing.T) {
	h := NewBaseAPIHandlers(nil, nil)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx := context.WithValue(context.Background(), "gin", c)
	if got := h.applyModelRouter(ctx, "claude", "claude-opus-5-5", nil, true, modelExecutionOptions{}); got.ScopedPrefix {
		t.Fatalf("root route changed: %+v", got)
	}
	reg := registry.GetGlobalRegistry()
	for _, provider := range []string{"devin", "custom-provider"} {
		id := t.Name() + provider
		reg.RegisterClient(id, provider, []*registry.ModelInfo{{ID: "team/gpt-6-1-sol"}})
		t.Cleanup(func() { reg.UnregisterClient(id) })
	}
	c.Set(ModelPrefixContextKey, "team")
	c.Set(ModelPrefixRoutesContextKey, map[string]string{"gpt-6.1-sol": "team/gpt-6-1-sol"})
	for _, model := range []string{"gpt-6.1-sol", "gpt-6.1-sol(xhigh)"} {
		decision := h.applyModelRouter(ctx, "openai", model, nil, true, modelExecutionOptions{})
		providers, target, err := h.providersForExecution(model, model, false, decision, modelExecutionOptions{})
		if err != nil || len(providers) != 2 {
			t.Fatalf("provider pool changed: %v %v", providers, err)
		}
		want := "team/gpt-6-1-sol"
		if model != "gpt-6.1-sol" {
			want += "(xhigh)"
		}
		if target != want {
			t.Fatalf("target=%s want %s", target, want)
		}
	}
	for _, model := range []string{"other/gpt-6.1-sol", "unknown"} {
		decision := h.applyModelRouter(ctx, "openai", model, nil, true, modelExecutionOptions{})
		providers, _, err := h.providersForExecution(model, model, false, decision, modelExecutionOptions{})
		if err == nil || len(providers) > 0 {
			t.Fatalf("request escaped prefix: %v %v", providers, err)
		}
	}
	for model, want := range map[string]string{"gpt-6.1-sol-thinking": "team/gpt-6-1-sol(high)", "gpt-6.1-sol(2048)": "team/gpt-6-1-sol(2048)"} {
		decision, _ := prefixedModelRoute(ctx, model)
		if decision.Model != want {
			t.Fatalf("suffix lost: %+v", decision)
		}
	}
	if got := reg.GetModelProviders("team/gpt-6-1-sol"); !reflect.DeepEqual(got, []string{"custom-provider", "devin"}) && len(got) != 2 {
		t.Fatal(got)
	}
}

func TestDevinRootThinkingSuffixKeepsCredentialSelectionOnBase(t *testing.T) {
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(t.Name(), "devin", []*registry.ModelInfo{{ID: "devin/claude-suffix-test"}})
	t.Cleanup(func() { reg.UnregisterClient(t.Name()) })
	h := NewBaseAPIHandlers(nil, nil)
	for suffix, want := range map[string]string{"-thinking": "high", "-think": "high", "-max": "max", "-none": "none"} {
		providers, model, err := h.getRequestDetails("devin/claude-suffix-test" + suffix)
		if err != nil || len(providers) != 1 || providers[0] != "devin" || model != "devin/claude-suffix-test("+want+")" {
			t.Fatalf("suffix=%s route=%v model=%s err=%v", suffix, providers, model, err)
		}
	}
	if got := normalizeDevinRequestSuffix("other/claude-suffix-test-thinking"); got != "other/claude-suffix-test-thinking" {
		t.Fatal("other provider changed")
	}
}
