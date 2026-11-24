# Module map

Devin executor: internal/runtime/executor/devin_executor.go. Devin helpers: internal/runtime/executor/helps/devin_*.go. Model catalog: internal/registry/models/devin_models.json. Shared translators are unchanged unless a Devin-specific integration requires them.

`internal/api/model_prefix_*` adds generic scoped model catalogs and HTTP routes. `sdk/api/handlers/model_prefix_route.go` resolves scoped public names to original registered IDs before existing provider selection. Devin runtime changes remain isolated to its executor/helpers.
