# Claim JSON compatibility fixtures

The `claim-*-origin-main.json` files were captured on 2026-10-20 from the
CLI built directly from local `origin/main` at
`0b18a235cc7e83189bfbc0ff465607d353e3e551`, before compact human output.
The historical tree was materialized with `git archive origin/main` and built
with `GOPROXY=off GOTOOLCHAIN=local go build`; no fetch or installation occurred.

Each capture used an independent temporary Git repository with `tracked.txt`
committed and `main` tracking the local `baseline` branch. JSON commands took
and released `older`, took and released `recent`, then took `current`.
The final command was `claim take/status/check/release --repo <fixture> --json`,
with `--slice current` except for status. No `--history` flag was supplied to
the historical CLI. Only RFC3339 UTC timestamps become `<TIME>` and the
elapsed `age_seconds` becomes zero. All other bytes remain the old CLI output.

`TestClaimHistoryJSONUnchanged` compares both current flag forms against these
same historical fixtures. Do not regenerate them from current output.
