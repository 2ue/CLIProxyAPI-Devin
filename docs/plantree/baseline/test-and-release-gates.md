# Test and release gates

Run targeted Devin regressions and service build. Verify real isolated clients plus SSE/tool-result round trips. Commit and push only to 2ue/CLIProxyAPI-Devin. Build on 87, canary before replacing the local Compose image, retain rollback.
