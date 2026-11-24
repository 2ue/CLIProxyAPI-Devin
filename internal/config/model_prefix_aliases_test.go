package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModelPrefixAliasesV8RoundTripAndClone(t *testing.T) {
	raw := []byte("config-version: 8\nrouting:\n  model-prefix-aliases:\n    team:\n      gpt-6.1-sol: vendor-name\n")
	if err := ValidateV8Config(raw); err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseConfigBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ModelPrefixAliases["team"]["gpt-6.1-sol"] != "vendor-name" {
		t.Fatal("mapping lost")
	}
	clone := cfg.CloneForRuntime()
	clone.ModelPrefixAliases["team"]["gpt-6.1-sol"] = "other"
	if cfg.ModelPrefixAliases["team"]["gpt-6.1-sol"] != "vendor-name" {
		t.Fatal("clone shared mapping")
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := SaveConfigPreserveComments(path, cfg); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseConfigBytes(saved)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.ModelPrefixAliases["team"]["gpt-6.1-sol"] != "vendor-name" {
		t.Fatal("save lost mapping")
	}
}

func TestDevinThinkingDisplayDefaultAndV8Option(t *testing.T) {
	cfg, err := ParseConfigBytes([]byte("config-version: 8\noauth:\n  providers:\n    devin:\n      show-thinking: true\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Devin.ShowThinking {
		t.Fatal("explicit display option lost")
	}
	defaults, err := ParseConfigBytes([]byte("config-version: 8\n"))
	if err != nil {
		t.Fatal(err)
	}
	if defaults.Devin.ShowThinking {
		t.Fatal("thought text must be hidden by default")
	}
}
