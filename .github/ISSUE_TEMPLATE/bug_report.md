---
name: Bug report
about: Something misbehaved — a crash, a wrong decode, a silent feed
labels: bug
---

## Environment (required)

- Platform / OS version (e.g. `x86_64-unknown-linux-gnu`, Ubuntu 22.04):
- `deshred --version`:
- Ingest mode: unicast (`SHRED_GROUPS` empty) / multicast
- `--rpc` / `SHRED_RPC` set: yes / no (degraded mode changes expected behavior — see docs/limits.md)
- License key set (`--license` / `SHRED_LICENSE`): yes / no — never paste the key itself

## What happened

<!-- Exact command, exact output. Quote errors verbatim. -->

## What you expected

## Logs / evidence

<!--
Relevant log lines (SHRED_LOG_LEVEL=debug helps).
For decode-accuracy claims: the strongest report is a `deshred verify` run
against a single archive endpoint — include its output.
Reminder: a CPI-wrapped swap being invisible is a documented structural limit
(docs/limits.md), not a bug.
-->
