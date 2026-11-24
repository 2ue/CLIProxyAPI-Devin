# Devin client compatibility

Mode: execute-ready for verified Devin repairs and the additive scoped entrypoint.

Objective: repair Devin-specific protocol defects so supported Claude, Gemini, and GPT models work in their native coding clients. Preserve existing credentials and unrelated providers.

Acceptance: real client reasoning request fields, streaming, file tools, MCP, and multi-turn results are checked. Smoke-test the advertised catalog without silently replacing models. Document upstream limitations rather than claiming universal support.

Read [roadmap](roadmap.md), [status](implementation-status.md), and [baseline](../../baseline/README.md).

Scoped entrypoint: `/llm/{prefix}` works for all registered model namespaces and coexists with the original root routes without a switch. Existing configured aliases are preserved. The vendor catalog reconciles platform punctuation with official identities; optional `routing.model-prefix-aliases` handles explicit scoped overrides. Requests resolve to the original full model ID and eligible provider pool. Authentication, body limits, text/tool integrity, protocol metadata and client reasoning suffixes are covered. See [HTTP route reference](../../../model-prefix-routes.md).
