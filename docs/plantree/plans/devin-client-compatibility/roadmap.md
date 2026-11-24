# Roadmap

## Done

- SDK identity metadata adaptation deployed as bad21ae3.
- All 31 Claude/Gemini/GPT models passed isolated native-client file reads.
- Representative builtin tools and MCP round trips passed in all three clients.
- Repaired Devin numeric/disabled/Gemini thinking controls, suffix credential routing, request-scoped refusals and model-not-found classification.
- Added generic `/llm/{prefix}` HTTP routes and official model identities without changing root IDs or existing aliases.
- Added optional scoped aliases and default-hidden Devin thought display with signature retention.
- Targeted tests, race checks, formatting, build, and final `go test -p 1 ./...` passed.
- Pushed eaa8df00, pulled and built on 87, passed canary and production inference.
- Six production native-client tool/MCP tasks passed. Original local configs and remote config/account JSON files were unchanged.

## In Progress

None.

## Next

- Observe upstream model availability and public network reliability during normal use.

## Deferred

- Scoped WebSockets/realtime/tunnels and unrelated upstream upgrades.
