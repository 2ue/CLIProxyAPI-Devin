# Verification checkpoint — 2026-11-24

Implementation commit: eaa8df0073cbb7029c7b0ca1ff69d6848a1854fa.

Production image: cliproxyapi-devin:eaa8df00. Healthy, restart count 0, no OOM. Root/scoped model lists each contain 53 models. Configuration/account JSON hashes matched before/after deployment. Runtime kv.db is intentionally mutable and is not a configuration checksum target.

Passed: final serial full repository tests, targeted Devin/prefix tests, route race checks, formatting and binary build. Parallel full runs hit existing timing-sensitive root socket/watcher tests; the final serial run passed.

All 31 model-family native file reads passed (14 Claude, 6 Gemini, 11 GPT). All three representative native clients passed builtin file tools and MCP. Six production cases passed via a temporary SSH forward to the same production service. Public direct requests showed intermittent connection timeouts; health and model requests also returned 200. No shared CLI configuration was edited; all eight original config hashes matched.

Devin thought-display test: hidden text 0 characters, signature 1012 characters; visible mode returned thought text and a correct answer. Thinking controls choose supported upstream variants. swe-1-6-slow remains unavailable upstream; prompt-dependent refusals and model instruction-following are distinct from protocol compatibility.

Evidence directory: `/Users/yuanfeijie/outputs/cpa-prefix-deploy-20261124`. Backup/rollback: `/root/docker-compose/cliproxyapi-59155/backups/model-prefix-20261124T065355Z/rollback.sh`.
