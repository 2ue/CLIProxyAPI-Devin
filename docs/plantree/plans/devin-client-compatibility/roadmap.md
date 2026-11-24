# Roadmap

## Done

- Existing SDK metadata adaptation deployed as bad21ae3.
- All 31 Claude/Gemini/GPT models completed isolated native-client file reads.
- Representative builtin tools and MCP round trips passed in all three native clients.
- Reproduced and repaired numeric thinking suffixes, explicit disabled controls, Gemini zero budgets, model-not-found classification, and request-scoped refusals.

## In Progress

- Generic `/llm/{prefix}` routes, canonical-name reconciliation, optional scoped aliases, and namespace restriction implemented.
- Scoped stream metadata, root compatibility, authentication, three native clients, tools/MCP and thought-display controls verified.
- Complete the final repository checks before publish/deploy.

## Next

- Push to the independent repository, build on 87, canary, deploy, and repeat production checks.

## Deferred

- Scoped WebSockets, shared-provider refactors, and upstream upgrades.
