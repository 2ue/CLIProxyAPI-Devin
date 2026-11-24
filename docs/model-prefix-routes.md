# Model prefix routes

`/llm/{prefix}` is an additive HTTP API entrypoint for every registered model prefix. It is always available alongside the existing root endpoints; no mode switch or provider-specific opt-in is required.

For example, root `/v1/models` retains `devin/gpt-6-1-sol`. `/llm/devin/v1/models` advertises `gpt-6.1-sol`, and requests with that model execute the original registered route. `/llm/team-a` works the same way for models published as `team-a/...`, including custom OpenAI-compatible providers.

## Client endpoints

| Client/protocol | Base URL | Model example |
| --- | --- | --- |
| Claude Code / Messages | `http://host:port/llm/devin` | `claude-opus-5-5` |
| Gemini CLI / GenerateContent | `http://host:port/llm/devin` | `gemini-3.8-flash` |
| Codex / OpenAI Responses | `http://host:port/llm/devin/v1` | `gpt-6.1-sol` |
| OpenAI Chat Completions | `http://host:port/llm/devin/v1` | `gpt-6.1-sol` |

The prefix identifies the existing model namespace, not necessarily a provider name. A provider without prefixed registered models has no scoped catalog until a prefix is configured for its credentials. Models already published without a prefix remain available on the original root routes.

Scoped HTTP routes include model lists/details, Chat Completions, legacy Completions, Claude Messages/count_tokens, Responses/compact, Gemini models/actions/Interactions, image generation/edits, and audio speech. Each operation still requires support from the selected provider. WebSockets, realtime sessions, tunnels, live endpoints, management and OAuth callbacks keep their existing root endpoints.

## Model identities and aliases

Existing explicitly configured aliases take precedence. Otherwise the scoped catalog reuses the authoritative Claude/Gemini/OpenAI/Kimi/XAI/Meta model catalog to reconcile punctuation differences. For example, `gpt-6-1-sol` becomes `gpt-6.1-sol`. The old Devin spelling `claude-5-fable` resolves to `claude-fable-5`. Exact canonical registrations win when multiple existing IDs resolve to the same name. Unknown identities remain unchanged after removing the namespace; the proxy does not infer a different model from a marketing label.

Optional overrides apply only to the scoped entrypoint:

```yaml
routing:
  model-prefix-aliases:
    my-platform:
      gpt-6.1-sol: vendor-deployment-name
```

The target must already exist as `my-platform/vendor-deployment-name`; overrides cannot create routes or escape the selected namespace. Existing root names, credential prefixes, upstream model aliases, cooldowns and scheduler choices are preserved. Parenthesized effort/budget suffixes are preserved. Suffixes such as `-thinking`/`-think` select high effort through CPA parentheses, and `-low`/`-high`/`-none` select their corresponding effort, so credential selection still uses the registered base model.

Model metadata in JSON/SSE responses uses the public scoped identity. Text, reasoning, signatures, tool arguments and tool results are not searched or replaced. Authentication, request size limits and example-key safe mode apply to the new routes. No credentials are added or changed by enabling this capability.

## Sources

Canonical names come from the existing CPA vendor catalogs. Additional retained OpenAI IDs were verified against https://developers.openai.com/api/docs/models, including `gpt-5.4`, `gpt-5.4-mini`, `gpt-5.5`, and https://developers.openai.com/api/docs/models/gpt-6.1-sol. This feature does not replace the deployed provider's context limits or model availability with official API limits.

## Devin thought display

Devin upstream reasoning remains enabled according to the request controls even when thought text is hidden. By default, `oauth.providers.devin.show-thinking` is false. Set it to true to expose upstream thought text. Signatures, tool arguments, and answers remain intact in both modes.

```yaml
oauth:
  providers:
    devin:
      show-thinking: true
```
